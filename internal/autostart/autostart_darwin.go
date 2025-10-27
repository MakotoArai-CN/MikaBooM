//go:build darwin

package autostart

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func GetAutostartPath() (string, error) {
	manager, err := NewAutostartManager()
	if err != nil {
		return "", err
	}

	launchAgentsDir, err := manager.getMacOSLaunchAgentsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(launchAgentsDir, "com.mikaboom.resourcemonitor.plist"), nil
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
`, m.escapeXML(m.appPath),
		m.escapeXML(m.configPath),
		m.escapeXML(m.appDir),
		m.escapeXML(filepath.Join(m.appDir, "logs", "stdout.log")),
		m.escapeXML(filepath.Join(m.appDir, "logs", "stderr.log")))
}

func (m *AutostartManager) escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// Windows 和 Linux 的 stub 方法（macOS 编译时需要）
func (m *AutostartManager) enableWindows() error {
	return fmt.Errorf("不支持的操作系统")
}

func (m *AutostartManager) disableWindows() error {
	return fmt.Errorf("不支持的操作系统")
}

func (m *AutostartManager) isEnabledWindows() (bool, error) {
	return false, fmt.Errorf("不支持的操作系统")
}

func (m *AutostartManager) enableLinux() error {
	return fmt.Errorf("不支持的操作系统")
}

func (m *AutostartManager) disableLinux() error {
	return fmt.Errorf("不支持的操作系统")
}

func (m *AutostartManager) isEnabledLinux() (bool, error) {
	return false, fmt.Errorf("不支持的操作系统")
}