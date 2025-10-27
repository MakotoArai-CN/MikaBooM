//go:build windows

package autostart

import (
	"fmt"
	"log"

	"golang.org/x/sys/windows/registry"
)

func GetAutostartPath() (string, error) {
	manager, err := NewAutostartManager()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Run\\%s", manager.appName), nil
}

func (m *AutostartManager) enableWindows() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("打开注册表失败: %w", err)
	}
	defer key.Close()

	startupCmd := fmt.Sprintf("\"%s\" -window=false -c \"%s\"", m.appPath, m.configPath)
	err = key.SetStringValue(m.appName, startupCmd)
	if err != nil {
		return fmt.Errorf("设置注册表值失败: %w", err)
	}

	log.Printf("✓ Windows 自启动已启用: %s", startupCmd)
	return nil
}

func (m *AutostartManager) disableWindows() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.ALL_ACCESS)
	if err != nil {
		log.Printf("⚠ 打开注册表失败（可能已经不存在）: %v", err)
		return nil
	}
	defer key.Close()

	err = key.DeleteValue(m.appName)
	if err != nil {
		if err == registry.ErrNotExist {
			log.Println("✓ Windows 自启动已禁用（注册表值不存在）")
			return nil
		}
		return fmt.Errorf("删除注册表值失败: %w", err)
	}

	log.Println("✓ Windows 自启动已禁用")
	return nil
}

func (m *AutostartManager) isEnabledWindows() (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE)
	if err != nil {
		return false, nil
	}
	defer key.Close()

	_, _, err = key.GetStringValue(m.appName)
	if err != nil {
		if err == registry.ErrNotExist {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// Linux 和 macOS 的 stub 方法（Windows 编译时需要）
func (m *AutostartManager) enableLinux() error {
	return fmt.Errorf("不支持的操作系统")
}

func (m *AutostartManager) disableLinux() error {
	return fmt.Errorf("不支持的操作系统")
}

func (m *AutostartManager) isEnabledLinux() (bool, error) {
	return false, fmt.Errorf("不支持的操作系统")
}

func (m *AutostartManager) enableMacOS() error {
	return fmt.Errorf("不支持的操作系统")
}

func (m *AutostartManager) disableMacOS() error {
	return fmt.Errorf("不支持的操作系统")
}

func (m *AutostartManager) isEnabledMacOS() (bool, error) {
	return false, fmt.Errorf("不支持的操作系统")
}