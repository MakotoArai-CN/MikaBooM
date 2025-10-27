package autostart

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type AutostartManager struct {
	appName     string
	appPath     string
	appDir      string
	configPath  string
	displayName string
	description string
}

func NewAutostartManager() (*AutostartManager, error) {
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("获取可执行文件路径失败: %w", err)
	}

	realPath, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		realPath = exePath
	}

	appDir := filepath.Dir(realPath)
	configPath := filepath.Join(appDir, "config.yaml")

	return &AutostartManager{
		appName:     "MikaBooM",
		appPath:     realPath,
		appDir:      appDir,
		configPath:  configPath,
		displayName: "Resource Monitor",
		description: "System Resource Monitor and Adjuster - Miku Edition",
	}, nil
}

// 公共接口函数
func Enable() error {
	manager, err := NewAutostartManager()
	if err != nil {
		return err
	}
	return manager.Enable()
}

func Disable() error {
	manager, err := NewAutostartManager()
	if err != nil {
		return err
	}
	return manager.Disable()
}

func IsEnabled() (bool, error) {
	manager, err := NewAutostartManager()
	if err != nil {
		return false, err
	}
	return manager.IsEnabled()
}

func ValidateAutostartSetup() error {
	enabled, err := IsEnabled()
	if err != nil {
		return fmt.Errorf("检查自启动状态失败: %w", err)
	}

	if !enabled {
		return fmt.Errorf("自启动未启用")
	}

	return nil
}

func CleanupAutostartFiles() error {
	return Disable()
}

// 平台特定方法的接口（由平台特定文件实现）
func (m *AutostartManager) Enable() error {
	switch runtime.GOOS {
	case "windows":
		return m.enableWindows()
	case "linux":
		return m.enableLinux()
	case "darwin":
		return m.enableMacOS()
	default:
		return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}

func (m *AutostartManager) Disable() error {
	switch runtime.GOOS {
	case "windows":
		return m.disableWindows()
	case "linux":
		return m.disableLinux()
	case "darwin":
		return m.disableMacOS()
	default:
		return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}

func (m *AutostartManager) IsEnabled() (bool, error) {
	switch runtime.GOOS {
	case "windows":
		return m.isEnabledWindows()
	case "linux":
		return m.isEnabledLinux()
	case "darwin":
		return m.isEnabledMacOS()
	default:
		return false, fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}