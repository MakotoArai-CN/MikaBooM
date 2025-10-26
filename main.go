package main

import (
	"MikaBooM/internal/autostart"
	"MikaBooM/internal/config"
	"MikaBooM/internal/monitor"
	"MikaBooM/internal/notify"
	"MikaBooM/internal/sysinfo"
	"MikaBooM/internal/tray"
	"MikaBooM/internal/version"
	"MikaBooM/internal/worker"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatih/color"
)

var (
	cpuThreshold     = flag.Int("cpu", -1, "CPU占用率阈值 (0-100)")
	memThreshold     = flag.Int("mem", -1, "内存占用率阈值 (0-100)")
	enableAutoStart  = flag.Bool("auto", false, "启用自启动")
	disableAutoStart = flag.Bool("noauto", false, "禁用自启动")
	showWindow       = flag.String("window", "", "显示窗口 (true/false)")
	showVersion      = flag.Bool("v", false, "显示版本信息")
	showHelp         = flag.Bool("h", false, "显示帮助信息")
	configFile       = flag.String("c", "", "指定配置文件路径")
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("程序异常恢复: %v", r)
		}
	}()

	flag.Parse()

	if *showHelp {
		showHelpInfo()
		return
	}

	if *showVersion {
		version.ShowVersion()
		return
	}

	// 查找配置文件
	cfgPath, err := config.FindConfigFile(*configFile)
	if err != nil {
		color.Red("✗ 查找配置文件失败: %v", err)
		log.Fatalf("查找配置文件失败: %v", err)
	}

	// 检查配置文件是否存在
	configExists := config.ConfigExists(cfgPath)
	
	// 加载配置（如果不存在会自动创建）
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		color.Red("✗ 加载配置文件失败: %v", err)
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 验证配置
	if err := config.ValidateConfig(cfg); err != nil {
		color.Red("✗ 配置验证失败: %v", err)
		log.Fatalf("配置验证失败: %v", err)
	}

	// 显示配置文件信息
	if !configExists {
		color.Green("✓ 配置文件不存在，已创建默认配置文件")
		color.Cyan("  路径: %s", cfgPath)
	} else {
		if *configFile != "" {
			color.Cyan("📝 使用指定的配置文件: %s", cfgPath)
		} else {
			color.Cyan("📝 使用配置文件: %s", cfgPath)
		}
	}

	// 命令行参数优先级高于配置文件
	if *cpuThreshold >= 0 {
		cfg.CPUThreshold = *cpuThreshold
		color.Yellow("⚙️  命令行参数覆盖: CPU阈值 = %d%%", cfg.CPUThreshold)
	}
	if *memThreshold >= 0 {
		cfg.MemoryThreshold = *memThreshold
		color.Yellow("⚙️  命令行参数覆盖: 内存阈值 = %d%%", cfg.MemoryThreshold)
	}
	if *showWindow != "" {
		switch *showWindow {
		case "true", "1", "yes", "on":
			cfg.ShowWindow = true
			color.Yellow("⚙️  命令行参数覆盖: 显示窗口 = true")
		case "false", "0", "no", "off":
			cfg.ShowWindow = false
			color.Yellow("⚙️  命令行参数覆盖: 显示窗口 = false")
		default:
			color.Yellow("⚠ 无效的 -window 参数值: %s (使用 true 或 false)", *showWindow)
		}
	}

	// 处理自启动设置
	if *enableAutoStart {
		if err := autostart.Enable(); err != nil {
			color.Red("✗ 启用自启动失败: %v", err)
			if cfg.ShowWindow {
				log.Printf("错误详情: %v", err)
			}
		} else {
			color.Green("✓ 自启动已启用")
			if cfg.ShowWindow {
				path, _ := autostart.GetAutostartPath()
				color.Cyan("  路径: %s", path)
			}
		}
		return
	}

	if *disableAutoStart {
		if err := autostart.Disable(); err != nil {
			color.Red("✗ 禁用自启动失败: %v", err)
		} else {
			color.Green("✓ 自启动已禁用")
		}
		return
	}

	// 如果配置文件中设置了自启动，且当前未启用，则自动启用
	if cfg.AutoStart {
		enabled, err := autostart.IsEnabled()
		if err != nil {
			if cfg.ShowWindow {
				color.Yellow("⚠ 检查自启动状态失败: %v", err)
			}
		} else if !enabled {
			if err := autostart.Enable(); err != nil {
				if cfg.ShowWindow {
					color.Yellow("⚠ 自动启用自启动失败: %v", err)
				}
			} else {
				if cfg.ShowWindow {
					color.Green("✓ 已根据配置自动启用自启动")
				}
			}
		}
	}

	// 设置日志输出
	if !cfg.ShowWindow {
		log.SetOutput(io.Discard)
	}

	// 显示欢迎信息
	if cfg.ShowWindow {
		fmt.Println() // 空行分隔
		showWelcome(cfg)
	}

	// 检查版本有效性
	versionValid := version.IsValid()
	if !versionValid {
		if cfg.ShowWindow {
			color.Yellow("╔═══════════════════════════════════════════════╗")
			color.Yellow("║           ⚠️  程序配置故障 ⚠️                  ║")
			color.Yellow("║   更新程序可能解决，请更新应用程序             ║")
			color.Yellow("║   当前仅支持监控功能，计算功能已停止           ║")
			color.Yellow("╚═══════════════════════════════════════════════╝")
			fmt.Println()
		}
		cfg.EnableWorker = false
	}

	// 初始化监控器
	cpuMonitor := monitor.NewCPUMonitor()
	memMonitor := monitor.NewMemoryMonitor()
	notifier := notify.NewNotifier(cfg.Notification.Enabled, cfg.Notification.Cooldown)

	// 初始化工作器（如果版本有效）
	var cpuWorker *worker.CPUWorker
	var memWorker *worker.MemoryWorker
	if versionValid {
		cpuWorker = worker.NewCPUWorker(cfg.CPUThreshold)
		memWorker = worker.NewMemoryWorker(cfg.MemoryThreshold)

		totalMem, err := memMonitor.GetTotalMemory()
		if err == nil {
			memWorker.SetTotalMemory(totalMem)
		}
	}

		go func() {
		tray.Start(cfg, cpuMonitor, memMonitor, cpuWorker, memWorker)
	}()

	ticker := time.NewTicker(time.Duration(cfg.UpdateInterval) * time.Second)
	defer ticker.Stop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	// 获取托盘退出信号 channel
	trayQuitChan := tray.GetQuitChannel()

	if cfg.ShowWindow {
		color.Cyan("🚀 程序启动成功，开始监控...")
		color.Cyan("📺 窗口显示: 已启用")
		color.Cyan("⏱️  更新间隔: %d 秒", cfg.UpdateInterval)
		fmt.Println()
	}

	lastCPUWorkerState := false
	lastMemWorkerState := false

	for {
		select {
		case <-ticker.C:
			cpuUsage, err := cpuMonitor.GetUsage()
			if err != nil {
				if cfg.ShowWindow {
					color.Red("✗ 获取CPU使用率失败: %v", err)
				}
				continue
			}

			memUsage, err := memMonitor.GetUsage()
			if err != nil {
				if cfg.ShowWindow {
					color.Red("✗ 获取内存使用率失败: %v", err)
				}
				continue
			}

			if cfg.ShowWindow {
				displayMonitorInfo(cpuUsage, memUsage, cfg, cpuWorker, memWorker)
			}

			if versionValid && cpuWorker != nil && memWorker != nil {
				cpuWorkerUsage := cpuWorker.GetUsage()
				memWorkerUsage := memWorker.GetUsage()

				otherCPUUsage := cpuUsage - cpuWorkerUsage
				otherMemUsage := memUsage - memWorkerUsage

				if otherCPUUsage < 0 {
					otherCPUUsage = 0
				}
				if otherMemUsage < 0 {
					otherMemUsage = 0
				}

				shouldCPUWork := otherCPUUsage < float64(cfg.CPUThreshold)
				if shouldCPUWork != lastCPUWorkerState {
					if shouldCPUWork {
						cpuWorker.Start()
						if cfg.ShowWindow {
							color.Green("✓ [CPU] 其他程序占用 %.1f%% < 阈值 %d%%，开始CPU计算", otherCPUUsage, cfg.CPUThreshold)
						}
						notifier.NotifyCPUWorkStart(cfg.CPUThreshold)
					} else {
						cpuWorker.Stop()
						if cfg.ShowWindow {
							color.Yellow("⚠ [CPU] 其他程序占用 %.1f%% >= 阈值 %d%%，停止CPU计算", otherCPUUsage, cfg.CPUThreshold)
						}
						notifier.NotifyCPUWorkStop()
					}
					lastCPUWorkerState = shouldCPUWork
				}

				if shouldCPUWork {
					targetCPUWorkerUsage := float64(cfg.CPUThreshold) - otherCPUUsage
					if targetCPUWorkerUsage < 0 {
						targetCPUWorkerUsage = 0
					}
					if targetCPUWorkerUsage > 100 {
						targetCPUWorkerUsage = 100
					}
					cpuWorker.AdjustLoad(cpuWorkerUsage, targetCPUWorkerUsage)
				}

				shouldMemWork := otherMemUsage < float64(cfg.MemoryThreshold)
				if shouldMemWork != lastMemWorkerState {
					if shouldMemWork {
						memWorker.Start()
						if cfg.ShowWindow {
							color.Green("✓ [MEM] 其他程序占用 %.1f%% < 阈值 %d%%，开始内存计算", otherMemUsage, cfg.MemoryThreshold)
						}
						notifier.NotifyMemWorkStart(cfg.MemoryThreshold)
					} else {
						memWorker.Stop()
						if cfg.ShowWindow {
							color.Yellow("⚠ [MEM] 其他程序占用 %.1f%% >= 阈值 %d%%，停止内存计算", otherMemUsage, cfg.MemoryThreshold)
						}
						notifier.NotifyMemWorkStop()
					}
					lastMemWorkerState = shouldMemWork
				}

				if shouldMemWork {
					targetMemWorkerUsage := float64(cfg.MemoryThreshold) - otherMemUsage
					if targetMemWorkerUsage < 0 {
						targetMemWorkerUsage = 0
					}
					if targetMemWorkerUsage > 80 {
						targetMemWorkerUsage = 80
					}
					memWorker.AdjustLoad(memWorkerUsage, targetMemWorkerUsage)
				}
			}

		case <-sigChan:
			// 收到系统信号（Ctrl+C）
			if cfg.ShowWindow {
				color.Cyan("📡 接收到退出信号，正在清理...")
			}
			if cpuWorker != nil {
				cpuWorker.Stop()
			}
			if memWorker != nil {
				memWorker.Stop()
			}
			if cfg.ShowWindow {
				color.Green("✓ 程序已安全退出")
			}
			return

		case <-trayQuitChan:
			// 收到托盘退出信号
			if cfg.ShowWindow {
				color.Cyan("📡 从托盘接收到退出信号，正在清理...")
			}
			if cpuWorker != nil {
				cpuWorker.Stop()
			}
			if memWorker != nil {
				memWorker.Stop()
			}
			if cfg.ShowWindow {
				color.Green("✓ 程序已安全退出")
			}
			return
		}
	}
}

func showWelcome(cfg *config.Config) {
	cyan := color.New(color.FgCyan, color.Bold)
	cyan.Println("╔════════════════════════════════════════╗")
	cyan.Println("║    Resource Monitor - Miku Edition     ║")
	cyan.Println("╚════════════════════════════════════════╝")

	info := sysinfo.GetSystemInfo()
	color.New(color.FgHiMagenta).Printf("💻 系统: %s\n", info.OS)
	color.New(color.FgHiMagenta).Printf("🔧 CPU: %s (%d 核心)\n", info.CPUModel, info.CPUCores)
	color.New(color.FgHiMagenta).Printf("💾 内存: %.2f GB\n", float64(info.TotalMemory)/1024/1024/1024)
	fmt.Println()

	color.New(color.FgHiCyan).Printf("⚙️  CPU阈值: %d%%\n", cfg.CPUThreshold)
	color.New(color.FgHiCyan).Printf("⚙️  内存阈值: %d%%\n", cfg.MemoryThreshold)
	color.New(color.FgHiCyan).Printf("⚙️  窗口模式: %s\n", getWindowModeText(cfg.ShowWindow))

	// 显示自启动状态
	enabled, err := autostart.IsEnabled()
	if err == nil {
		if enabled {
			color.New(color.FgHiGreen).Printf("⚙️  自启动: 已启用 ✓\n")
		} else {
			color.New(color.FgHiYellow).Printf("⚙️  自启动: 未启用\n")
		}
	}

	fmt.Println()
}

func getWindowModeText(showWindow bool) string {
	if showWindow {
		return "显示 (前台运行)"
	}
	return "隐藏 (后台运行)"
}

func displayMonitorInfo(cpuUsage, memUsage float64, cfg *config.Config, cpuWorker *worker.CPUWorker, memWorker *worker.MemoryWorker) {
	timestamp := time.Now().Format("15:04:05")

	cpuColor := color.New(color.FgGreen)
	if cpuUsage > float64(cfg.CPUThreshold) {
		cpuColor = color.New(color.FgRed)
	} else if cpuUsage > float64(cfg.CPUThreshold)*0.8 {
		cpuColor = color.New(color.FgYellow)
	}

	memColor := color.New(color.FgGreen)
	if memUsage > float64(cfg.MemoryThreshold) {
		memColor = color.New(color.FgRed)
	} else if memUsage > float64(cfg.MemoryThreshold)*0.8 {
		memColor = color.New(color.FgYellow)
	}

	fmt.Printf("[%s] ", timestamp)
	cpuColor.Printf("CPU: %5.1f%% ", cpuUsage)
	memColor.Printf("MEM: %5.1f%%", memUsage)

	if cpuWorker != nil {
		if cpuWorker.IsRunning() {
			color.New(color.FgCyan).Printf(" [CPU-W: ON, I:%d%%]", cpuWorker.GetIntensity())
		} else {
			color.New(color.FgHiBlack).Printf(" [CPU-W: OFF]")
		}
	}

	if memWorker != nil {
		if memWorker.IsRunning() {
			allocMB := memWorker.GetAllocatedSize() / 1024 / 1024
			color.New(color.FgMagenta).Printf(" [MEM-W: ON, A:%dMB]", allocMB)
		} else {
			color.New(color.FgHiBlack).Printf(" [MEM-W: OFF]")
		}
	}

	fmt.Println()
}

func showHelpInfo() {
	cyan := color.New(color.FgCyan, color.Bold)
	cyan.Println("╔════════════════════════════════════════════════════════════╗")
	cyan.Println("║     Resource Monitor - 系统资源监控与调整工具              ║")
	cyan.Println("║              Miku Edition v1.0.0                           ║")
	cyan.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	color.New(color.FgYellow, color.Bold).Println("📖 用法:")
	fmt.Println("  MikaBooM [选项]")
	fmt.Println()

	color.New(color.FgYellow, color.Bold).Println("⚙️  选项:")
	fmt.Println("  -cpu <value>        设置CPU占用率阈值 (0-100)")
	fmt.Println("                      示例: -cpu 80")
	fmt.Println()
	fmt.Println("  -mem <value>        设置内存占用率阈值 (0-100)")
	fmt.Println("                      示例: -mem 70")
	fmt.Println()
	fmt.Println("  -window <value>     设置窗口显示模式")
	fmt.Println("                      true/1/yes/on  - 显示窗口（前台运行）")
	fmt.Println("                      false/0/no/off - 隐藏窗口（后台运行）")
	fmt.Println("                      示例: -window=false")
	fmt.Println()
	fmt.Println("  -auto               启用开机自启动")
	fmt.Println()
	fmt.Println("  -noauto             禁用开机自启动")
	fmt.Println()
	fmt.Println("  -c <file>           指定配置文件路径")
	fmt.Println("                      支持绝对路径和相对路径")
	fmt.Println("                      未指定时自动使用可执行文件同级目录下的 config.yaml")
	fmt.Println("                      示例: -c /path/to/config.yaml")
	fmt.Println()
	fmt.Println("  -v                  显示版本信息")
	fmt.Println()
	fmt.Println("  -h                  显示此帮助信息")
	fmt.Println()

	color.New(color.FgGreen, color.Bold).Println("💡 示例:")
	fmt.Println("  # 首次运行（自动创建配置文件）")
	fmt.Println("  MikaBooM")
	fmt.Println()
	fmt.Println("  # 使用默认配置文件，设置CPU阈值80%")
	fmt.Println("  MikaBooM -cpu 80")
	fmt.Println()
	fmt.Println("  # 后台运行，不显示窗口")
	fmt.Println("  MikaBooM -window=false")
	fmt.Println()
	fmt.Println("  # 启用开机自启动")
	fmt.Println("  MikaBooM -auto")
	fmt.Println()
	fmt.Println("  # 禁用开机自启动")
	fmt.Println("  MikaBooM -noauto")
	fmt.Println()
	fmt.Println("  # 使用自定义配置文件")
	fmt.Println("  MikaBooM -c /path/to/custom-config.yaml")
	fmt.Println()
	fmt.Println("  # 使用相对路径的配置文件")
	fmt.Println("  MikaBooM -c configs/server.yaml")
	fmt.Println()
	fmt.Println("  # 查看版本信息")
	fmt.Println("  MikaBooM -v")
	fmt.Println()

	color.New(color.FgCyan, color.Bold).Println("📝 配置文件:")
	fmt.Println("  默认位置: 可执行文件同级目录下的 config.yaml")
	fmt.Println()
	fmt.Println("  配置文件查找规则:")
	fmt.Println("    1. 如果使用 -c 参数指定了配置文件，使用指定的文件")
	fmt.Println("    2. 否则，在可执行文件同级目录查找 config.yaml")
	fmt.Println("    3. 如果配置文件不存在，自动创建默认配置文件")
	fmt.Println()
	fmt.Println("  支持的配置项:")
	fmt.Println("    - cpu_threshold      CPU阈值 (0-100)")
	fmt.Println("    - memory_threshold   内存阈值 (0-100)")
	fmt.Println("    - show_window        是否显示窗口 (true/false)")
	fmt.Println("    - auto_start         是否自启动 (true/false)")
	fmt.Println("    - update_interval    更新间隔（秒）")
	fmt.Println("    - notification       通知设置")
	fmt.Println("      - enabled          是否启用通知")
	fmt.Println("      - cooldown         通知冷却时间（秒）")
	fmt.Println()

	color.New(color.FgMagenta, color.Bold).Println("🔧 配置优先级:")
	fmt.Println("  命令行参数 > 配置文件")
	fmt.Println("  示例: 配置文件中 cpu_threshold=70，命令行使用 -cpu 80")
	fmt.Println("        最终使用 CPU阈值=80%")
	fmt.Println()

	color.New(color.FgCyan, color.Bold).Println("📂 配置文件示例:")
	fmt.Println(`  cpu_threshold: 70
  memory_threshold: 70
  auto_start: true
  show_window: true
  update_interval: 2
  notification:
    enabled: true
    cooldown: 60`)
	fmt.Println()

	color.New(color.FgMagenta).Println("👤 作者: Makoto")
	color.New(color.FgCyan).Println("📧 项目: MikaBooM - Resource Monitor Miku Edition")
	fmt.Println()
}