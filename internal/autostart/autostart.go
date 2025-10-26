package autostart

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/sys/windows/registry"
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
	// 修复：使用 ALL_ACCESS 权限打开注册表，确保可以删除值
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.ALL_ACCESS)
	if err != nil {
		// 如果打开失败，可能是权限问题或注册表项不存在
		log.Printf("⚠ 打开注册表失败（可能已经不存在）: %v", err)
		return nil
	}
	defer key.Close()

	// 删除注册表值
	err = key.DeleteValue(m.appName)
	if err != nil {
		if err == registry.ErrNotExist {
			// 值不存在，认为删除成功
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
`,
		m.displayName,
		m.description,
		m.appPath,
		m.configPath,
		m.appDir,
	)
}

func (m *AutostartManager) enableMacOS() error {
	launchAgentsDir, err := m.getMacOSLaunchAgentsDir()
	if err != nil {
		return fmt.Errorf("获取LaunchAgents目录失败: %w", err)
	}

	if err := os.MkdirAll(launchAgentsDir, 0755); err != nil {
		return fmt.Errorf("创建LaunchAgents目录失败: %w", err)
	}

	plistFile := filepath.Join(launchAgentsDir, "com.mikaboom.resourcemonitor.plist")
	plistContent := m.generateMacOSPlist()

	if err := os.WriteFile(plistFile, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("创建plist文件失败: %w", err)
	}

	log.Printf("✓ macOS 自启动已启用: %s", plistFile)
	return nil
}

func (m *AutostartManager) disableMacOS() error {
	launchAgentsDir, err := m.getMacOSLaunchAgentsDir()
	if err != nil {
		return fmt.Errorf("获取LaunchAgents目录失败: %w", err)
	}

	plistFile := filepath.Join(launchAgentsDir, "com.mikaboom.resourcemonitor.plist")

	if err := os.Remove(plistFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除plist文件失败: %w", err)
	}

	log.Println("✓ macOS 自启动已禁用")
	return nil
}

func (m *AutostartManager) isEnabledMacOS() (bool, error) {
	launchAgentsDir, err := m.getMacOSLaunchAgentsDir()
	if err != nil {
		return false, err
	}

	plistFile := filepath.Join(launchAgentsDir, "com.mikaboom.resourcemonitor.plist")
	_, err = os.Stat(plistFile)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (m *AutostartManager) getMacOSLaunchAgentsDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("无法获取用户主目录: %w", err)
	}
	return filepath.Join(homeDir, "Library", "LaunchAgents"), nil
}

func (m *AutostartManager) generateMacOSPlist() string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.mikaboom.resourcemonitor</string>
	
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
		<string>-window=false</string>
		<string>-c</string>
		<string>%s</string>
	</array>
	
	<key>WorkingDirectory</key>
	<string>%s</string>
	
	<key>RunAtLoad</key>
	<true/>
	
	<key>KeepAlive</key>
	<false/>
	
	<key>ProcessType</key>
	<string>Interactive</string>
	
	<key>StandardOutPath</key>
	<string>%s</string>
	
	<key>StandardErrorPath</key>
	<string>%s</string>
	
	<key>EnvironmentVariables</key>
	<dict>
		<key>PATH</key>
		<string>/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
	</dict>
</dict>
</plist>
`,
		m.escapeXML(m.appPath),
		m.escapeXML(m.configPath),
		m.escapeXML(m.appDir),
		m.escapeXML(filepath.Join(m.appDir, "logs", "stdout.log")),
		m.escapeXML(filepath.Join(m.appDir, "logs", "stderr.log")),
	)
}

func (m *AutostartManager) escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

func GetAutostartPath() (string, error) {
	manager, err := NewAutostartManager()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "windows":
		return fmt.Sprintf("HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Run\\%s", manager.appName), nil
	case "linux":
		autostartDir, err := manager.getLinuxAutostartDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(autostartDir, "MikaBooM.desktop"), nil
	case "darwin":
		launchAgentsDir, err := manager.getMacOSLaunchAgentsDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(launchAgentsDir, "com.mikaboom.resourcemonitor.plist"), nil
	default:
		return "", fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
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