//go:build linux

package autostart

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func GetAutostartPath() (string, error) {
	manager, err := NewAutostartManager()
	if err != nil {
		return "", err
	}

	autostartDir, err := manager.getLinuxAutostartDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(autostartDir, "MikaBooM.desktop"), nil
}

func (m *AutostartManager) enableLinux() error {
	autostartDir, err := m.getLinuxAutostartDir()
	if err != nil {
		return fmt.Errorf("获取autostart目录失败: %w", err)
	}

	if err := os.MkdirAll(autostartDir, 0755); err != nil {
		return fmt.Errorf("创建autostart目录失败: %w", err)
	}

	desktopFile := filepath.Join(autostartDir, "MikaBooM.desktop")
	desktopContent := m.generateLinuxDesktopFile()

	if err := os.WriteFile(desktopFile, []byte(desktopContent), 0755); err != nil {
		return fmt.Errorf("创建desktop文件失败: %w", err)
	}

	log.Printf("✓ Linux 自启动已启用: %s", desktopFile)
	return nil
}

func (m *AutostartManager) disableLinux() error {
	autostartDir, err := m.getLinuxAutostartDir()
	if err != nil {
		return fmt.Errorf("获取autostart目录失败: %w", err)
	}

	desktopFile := filepath.Join(autostartDir, "MikaBooM.desktop")
	if err := os.Remove(desktopFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除desktop文件失败: %w", err)
	}

	log.Println("✓ Linux 自启动已禁用")
	return nil
}

func (m *AutostartManager) isEnabledLinux() (bool, error) {
	autostartDir, err := m.getLinuxAutostartDir()
	if err != nil {
		return false, err
	}

	desktopFile := filepath.Join(autostartDir, "MikaBooM.desktop")
	_, err = os.Stat(desktopFile)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

func (m *AutostartManager) getLinuxAutostartDir() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("无法获取用户主目录: %w", err)
		}
		configHome = filepath.Join(homeDir, ".config")
	}

	return filepath.Join(configHome, "autostart"), nil
}

func (m *AutostartManager) generateLinuxDesktopFile() string {
	return fmt.Sprintf(`[Desktop Entry]
Type=Application
Version=1.0
Name=%s
GenericName=Resource Monitor
Comment=%s
Exec="%s" -window=false -c "%s"
Path=%s
Icon=utilities-system-monitor
Terminal=false
Categories=System;Monitor;Utility;
Keywords=resource;monitor;cpu;memory;system;
StartupNotify=false
X-GNOME-Autostart-enabled=true
X-GNOME-Autostart-Delay=5
X-KDE-autostart-after=panel
X-MATE-Autostart-enabled=true
Hidden=false
NoDisplay=false
`, m.displayName, m.description, m.appPath, m.configPath, m.appDir)
}

// Windows 和 macOS 的 stub 方法（Linux 编译时需要）
func (m *AutostartManager) enableWindows() error {
	return fmt.Errorf("不支持的操作系统")
}

func (m *AutostartManager) disableWindows() error {
	return fmt.Errorf("不支持的操作系统")
}

func (m *AutostartManager) isEnabledWindows() (bool, error) {
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