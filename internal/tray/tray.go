package tray

import (
	_ "embed"
	"fmt"
	"log"
	"runtime"
	"time"

	"MikaBooM/internal/autostart"
	"MikaBooM/internal/config"
	"MikaBooM/internal/monitor"
	"MikaBooM/internal/worker"

	"github.com/getlantern/systray"
)

//go:embed assets/icon_windows.ico
var iconWindows []byte

//go:embed assets/icon_linux.png
var iconLinux []byte

//go:embed assets/icon_macos.png
var iconMacOS []byte

var (
	cpuMonitor *monitor.CPUMonitor
	memMonitor *monitor.MemoryMonitor
	cpuWorker  *worker.CPUWorker
	memWorker  *worker.MemoryWorker

	mCPU        *systray.MenuItem
	mMEM        *systray.MenuItem
	mCPUWorker  *systray.MenuItem
	mMemWorker  *systray.MenuItem
	mAutostart  *systray.MenuItem
	mQuit       *systray.MenuItem
	
	stopUpdateLoop chan struct{}
	quitChan       chan struct{}
)

// GetQuitChannel 返回退出信号 channel
func GetQuitChannel() <-chan struct{} {
	if quitChan == nil {
		quitChan = make(chan struct{})
	}
	return quitChan
}

func Start(
	config *config.Config,
	cpu *monitor.CPUMonitor,
	mem *monitor.MemoryMonitor,
	cpuW *worker.CPUWorker,
	memW *worker.MemoryWorker,
) {
	cpuMonitor = cpu
	memMonitor = mem
	cpuWorker = cpuW
	memWorker = memW

	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetTitle("MB")
	systray.SetTooltip("MikaBooM - Resource Monitor Miku Edition")

	iconData := getIcon()
	if len(iconData) > 0 {
		systray.SetIcon(iconData)
		log.Printf("✓ 托盘图标已加载 (%d bytes, %s 平台)", len(iconData), runtime.GOOS)
	} else {
		log.Println("⚠ 托盘图标加载失败，使用系统默认图标")
	}

	mCPU = systray.AddMenuItem("CPU: 0.0%", "CPU使用率")
	mCPU.Disable()

	mMEM = systray.AddMenuItem("MEM: 0.0%", "内存使用率")
	mMEM.Disable()

	systray.AddSeparator()

	if cpuWorker != nil {
		mCPUWorker = systray.AddMenuItem("CPU计算: 就绪", "CPU工作器状态")
		mCPUWorker.Disable()
	}

	if memWorker != nil {
		mMemWorker = systray.AddMenuItem("内存计算: 就绪", "内存工作器状态")
		mMemWorker.Disable()
	}

	systray.AddSeparator()

	mAutostart = systray.AddMenuItem("自启动", "开启/关闭自启动")
	updateAutostartMenuItem()

	systray.AddSeparator()

	mStatus := systray.AddMenuItem("状态: 监控中", "当前状态")
	mStatus.Disable()

	systray.AddSeparator()

	mQuit = systray.AddMenuItem("退出", "退出程序")

	// 初始化 channel
	stopUpdateLoop = make(chan struct{})
	if quitChan == nil {
		quitChan = make(chan struct{})
	}

	go updateLoop()
	go handleMenuEvents()
}

func handleMenuEvents() {
	for {
		select {
		case <-mAutostart.ClickedCh:
			toggleAutostart()
		case <-mQuit.ClickedCh:
			// 通知主程序退出
			if quitChan != nil {
				close(quitChan)
			}
			systray.Quit()
			return
		}
	}
}

func toggleAutostart() {
	enabled, err := autostart.IsEnabled()
	if err != nil {
		log.Printf("检查自启动状态失败: %v", err)
		return
	}

	if enabled {
		if err := autostart.Disable(); err != nil {
			log.Printf("禁用自启动失败: %v", err)
			mAutostart.SetTitle("自启动 (禁用失败)")
		} else {
			log.Println("自启动已禁用")
			mAutostart.Uncheck()
			mAutostart.SetTitle("自启动")
		}
	} else {
		if err := autostart.Enable(); err != nil {
			log.Printf("启用自启动失败: %v", err)
			mAutostart.SetTitle("自启动 (启用失败)")
		} else {
			log.Println("自启动已启用")
			mAutostart.Check()
			mAutostart.SetTitle("自启动 ✓")
		}
	}

	updateAutostartMenuItem()
}

func updateAutostartMenuItem() {
	if mAutostart == nil {
		return
	}

	enabled, err := autostart.IsEnabled()
	if err != nil {
		mAutostart.SetTitle("自启动 (状态未知)")
		return
	}

	if enabled {
		mAutostart.Check()
		mAutostart.SetTitle("自启动 ✓")
	} else {
		mAutostart.Uncheck()
		mAutostart.SetTitle("自启动")
	}
}

func onExit() {
	// 先停止 updateLoop
	if stopUpdateLoop != nil {
		close(stopUpdateLoop)
	}
	
	// 等待 updateLoop 退出
	time.Sleep(100 * time.Millisecond)
	
	// 停止 worker
	if cpuWorker != nil {
		cpuWorker.Stop()
	}
	if memWorker != nil {
		memWorker.Stop()
	}
}

func updateLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stopUpdateLoop:
			return
		case <-ticker.C:
			cpuUsage, err := cpuMonitor.GetUsage()
			if err != nil {
				log.Printf("获取CPU使用率失败: %v", err)
				continue
			}

			memUsage, err := memMonitor.GetUsage()
			if err != nil {
				log.Printf("获取内存使用率失败: %v", err)
				continue
			}

			if mCPU != nil {
				mCPU.SetTitle(fmt.Sprintf("CPU: %.1f%%", cpuUsage))
			}
			if mMEM != nil {
				mMEM.SetTitle(fmt.Sprintf("MEM: %.1f%%", memUsage))
			}

			if cpuWorker != nil && mCPUWorker != nil {
				if cpuWorker.IsRunning() {
					intensity := cpuWorker.GetIntensity()
					workerUsage := cpuWorker.GetUsage()
					mCPUWorker.SetTitle(fmt.Sprintf("CPU计算: 运行中 (强度:%d%%, 占用:%.1f%%)", intensity, workerUsage))
				} else {
					mCPUWorker.SetTitle("CPU计算: 已停止")
				}
			}

			if memWorker != nil && mMemWorker != nil {
				if memWorker.IsRunning() {
					allocMB := memWorker.GetAllocatedSize() / 1024 / 1024
					targetMB := memWorker.GetTargetSize() / 1024 / 1024
					workerUsage := memWorker.GetUsage()
					mMemWorker.SetTitle(fmt.Sprintf("内存计算: 运行中 (已分配:%dMB, 目标:%dMB, 占用:%.1f%%)", allocMB, targetMB, workerUsage))
				} else {
					mMemWorker.SetTitle("内存计算: 已停止")
				}
			}

			tooltip := fmt.Sprintf("MikaBooM - Miku Edition\nCPU: %.1f%% | MEM: %.1f%%", cpuUsage, memUsage)
			if cpuWorker != nil && cpuWorker.IsRunning() {
				tooltip += fmt.Sprintf("\nCPU工作器: ON (强度:%d%%)", cpuWorker.GetIntensity())
			}
			if memWorker != nil && memWorker.IsRunning() {
				allocMB := memWorker.GetAllocatedSize() / 1024 / 1024
				tooltip += fmt.Sprintf("\n内存工作器: ON (已分配:%dMB)", allocMB)
			}
			systray.SetTooltip(tooltip)
		}
	}
}

func getIcon() []byte {
	switch runtime.GOOS {
	case "windows":
		if len(iconWindows) > 0 {
			return iconWindows
		}
		log.Println("⚠ Windows 图标未嵌入")
		return []byte{}
	case "linux":
		if len(iconLinux) > 0 {
			return iconLinux
		}
		log.Println("⚠ Linux 图标未嵌入")
		return []byte{}
	case "darwin":
		if len(iconMacOS) > 0 {
			return iconMacOS
		}
		log.Println("⚠ macOS 图标未嵌入")
		return []byte{}
	default:
		log.Printf("⚠ 未知操作系统: %s", runtime.GOOS)
		return []byte{}
	}
}