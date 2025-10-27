package main

import (
	"MikaBooM/internal/autostart"
	"MikaBooM/internal/config"
	"MikaBooM/internal/monitor"
	"MikaBooM/internal/notify"
	"MikaBooM/internal/sysinfo"
	"MikaBooM/internal/tray"
	"MikaBooM/internal/updater"
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
	checkUpdate      = flag.Bool("update", false, "检查并更新到最新版本")
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

	// 处理更新
	if *checkUpdate {
		handleUpdate()
		return
	}

	cfgPath, err := config.FindConfigFile(*configFile)
	if err != nil {
		color.Red("✗ 查找配置文件失败: %v", err)
		log.Fatalf("查找配置文件失败: %v", err)
	}

	configExists := config.ConfigExists(cfgPath)

	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		color.Red("✗ 加载配置文件失败: %v", err)
		log.Fatalf("加载配置文件失败: %v", err)
	}

	if err := config.ValidateConfig(cfg); err != nil {
		color.Red("✗ 配置验证失败: %v", err)
		log.Fatalf("配置验证失败: %v", err)
	}

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

	if !cfg.ShowWindow {
		log.SetOutput(io.Discard)
	}

	if cfg.ShowWindow {
		fmt.Println()
		showWelcome(cfg)
	}

	// 启动时检查更新
	if cfg.UpdateCheck.Enabled && cfg.UpdateCheck.CheckOnStartup {
		checkUpdateOnStartup(cfg)
	}

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

	cpuMonitor := monitor.NewCPUMonitor()
	memMonitor := monitor.NewMemoryMonitor()
	notifier := notify.NewNotifier(cfg.Notification.Enabled, cfg.Notification.Cooldown)

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

// checkUpdateOnStartup 启动时检查更新
func checkUpdateOnStartup(cfg *config.Config) {
	upd := updater.NewUpdater(version.GetVersion())

	// 静默检查更新
	release, hasUpdate, err := upd.CheckUpdateSilent()
	if err != nil {
		// 静默失败，不显示错误
		return
	}

	if hasUpdate {
		if !cfg.UpdateCheck.SilentCheck {
			updater.ShowUpdateNotice(release, version.GetVersion())
		}
	} else {
		if !cfg.UpdateCheck.SilentCheck && cfg.ShowWindow {
			color.Green("✓ 当前已是最新版本 v%s", version.GetVersion())
			fmt.Println()
		}
	}
}

// handleUpdate 处理更新逻辑
func handleUpdate() {
	color.Cyan("╔════════════════════════════════════════════════╗")
	color.Cyan("║           🔄 MikaBooM 更新检查                  ║")
	color.Cyan("╚════════════════════════════════════════════════╝")
	fmt.Println()

	upd := updater.NewUpdater(version.GetVersion())

	// 检查更新
	color.Cyan("🔍 正在检查更新...")
	color.Cyan("📡 仓库地址: %s", updater.GitHubRepo)
	fmt.Println()

	release, hasUpdate, err := upd.CheckUpdate()
	if err != nil {
		color.Red("✗ 检查更新失败: %v", err)
		fmt.Println()
		color.Yellow("请检查:")
		color.Yellow("  1. 网络连接是否正常")
		color.Yellow("  2. 是否可以访问 GitHub")
		color.Yellow("  3. 防火墙是否拦截")
		os.Exit(1)
	}

	if !hasUpdate {
		color.Green("✓ 已是最新版本 v%s", version.GetVersion())
		fmt.Println()
		color.Cyan("当前版本信息:")
		color.Cyan("  版本号: v%s", version.GetVersion())
		color.Cyan("  编译日期: %s", version.GetBuildDate())
		color.Cyan("  有效期至: %s", version.GetExpireDate())
		fmt.Println()
		return
	}

	// 显示更新信息
	updater.ShowUpdateInfo(release, version.GetVersion())

	// 询问是否更新
	fmt.Print("是否立即更新？[Y/n]: ")
	var answer string
	fmt.Scanln(&answer)

	if answer != "" && answer != "Y" && answer != "y" && answer != "yes" {
		color.Yellow("已取消更新")
		return
	}

	fmt.Println()

	// 执行更新
	if err := upd.PerformUpdate(release); err != nil {
		color.Red("✗ 更新失败: %v", err)
		fmt.Println()
		color.Yellow("可能的原因:")
		color.Yellow("  1. 网络连接中断")
		color.Yellow("  2. 权限不足（尝试使用管理员权限运行）")
		color.Yellow("  3. 磁盘空间不足")
		os.Exit(1)
	}

	color.Green("╔════════════════════════════════════════════════╗")
	color.Green("║           ✅ 更新成功！                         ║")
	color.Green("╚════════════════════════════════════════════════╝")
	fmt.Println()

	// 询问是否重启
	fmt.Print("是否立即重启程序？[Y/n]: ")
	fmt.Scanln(&answer)

	if answer == "" || answer == "Y" || answer == "y" || answer == "yes" {
		if err := upd.Restart(); err != nil {
			color.Red("✗ 重启失败: %v", err)
			color.Yellow("请手动重启程序")
		}
		os.Exit(0)
	}

	color.Cyan("请手动重启程序以使用新版本")
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
	fmt.Println("  -update             检查并更新到最新版本")
	fmt.Println("                      从 GitHub 仓库自动下载并安装更新")
	fmt.Println("                      支持所有平台的自动更新")
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
	fmt.Println("  # 检查并更新到最新版本")
	fmt.Println("  MikaBooM -update")
	fmt.Println()
	fmt.Println("  # 查看版本信息")
	fmt.Println("  MikaBooM -v")
	fmt.Println()

	color.New(color.FgCyan, color.Bold).Println("🔄 更新功能:")
	fmt.Println("  使用 -update 参数可以自动检查并安装更新:")
	fmt.Println("    1. 从 GitHub 检测最新版本")
	fmt.Println("    2. 自动下载适配当前系统的版本")
	fmt.Println("    3. 安全替换当前可执行文件（自动备份）")
	fmt.Println("    4. 可选择立即重启程序")
	fmt.Println()
	fmt.Println("  启动时自动检查更新:")
	fmt.Println("    - 可在配置文件中设置 update_check.enabled")
	fmt.Println("    - 可设置 update_check.check_on_startup")
	fmt.Println("    - 可设置 update_check.silent_check（静默检查）")
	fmt.Println()
	fmt.Println("  更新过程安全可靠:")
	fmt.Println("    - 下载到内存临时文件")
	fmt.Println("    - 更新前自动备份原文件")
	fmt.Println("    - 更新失败自动回滚")
	fmt.Println("    - 更新后自动清理临时文件")
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
	fmt.Println("    - update_check       更新检查设置")
	fmt.Println("      - enabled          是否启用更新检查")
	fmt.Println("      - check_on_startup 是否启动时检查")
	fmt.Println("      - silent_check     是否静默检查")
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
    cooldown: 60
  update_check:
    enabled: true
    check_on_startup: true
    silent_check: false`)
	fmt.Println()

	color.New(color.FgMagenta).Println("👤 作者: Makoto")
	color.New(color.FgCyan).Println("📧 项目: MikaBooM - Resource Monitor Miku Edition")
	color.New(color.FgCyan).Println("🔗 GitHub: https://github.com/MakotoArai-CN/MikaBooM")
	fmt.Println()
}