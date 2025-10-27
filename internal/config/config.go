package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	CPUThreshold    int                `yaml:"cpu_threshold"`
	MemoryThreshold int                `yaml:"memory_threshold"`
	AutoStart       bool               `yaml:"auto_start"`
	ShowWindow      bool               `yaml:"show_window"`
	UpdateInterval  int                `yaml:"update_interval"`
	Notification    NotificationConfig `yaml:"notification"`
	UpdateCheck     UpdateCheckConfig  `yaml:"update_check"`
	EnableWorker    bool               `yaml:"-"`
}

type NotificationConfig struct {
	Enabled  bool `yaml:"enabled"`
	Cooldown int  `yaml:"cooldown"`
}

type UpdateCheckConfig struct {
	Enabled         bool `yaml:"enabled"`
	CheckOnStartup  bool `yaml:"check_on_startup"`
	AutoDownload    bool `yaml:"auto_download"`
	SilentCheck     bool `yaml:"silent_check"`
}

func GetDefaultConfig() *Config {
	return &Config{
		CPUThreshold:    70,
		MemoryThreshold: 70,
		AutoStart:       true,
		ShowWindow:      true,
		UpdateInterval:  2,
		Notification: NotificationConfig{
			Enabled:  true,
			Cooldown: 60,
		},
		UpdateCheck: UpdateCheckConfig{
			Enabled:        true,
			CheckOnStartup: true,
			AutoDownload:   false,
			SilentCheck:    false,
		},
		EnableWorker: true,
	}
}

func LoadConfig(filepath string) (*Config, error) {
	cfg := GetDefaultConfig()

	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		if err := SaveConfig(filepath, cfg); err != nil {
			return cfg, fmt.Errorf("创建默认配置文件失败: %w", err)
		}
		return cfg, nil
	}

	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return cfg, nil
}

func SaveConfig(filepath string, cfg *Config) error {
	dir := filepath[:len(filepath)-len(filepath[len(filepath)-1:])]
	if dir != "" {
		dir = filepath[:len(filepath)-len(filepath[len(filepath)-1:])]
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建配置目录失败: %w", err)
		}
	}

	data := generateConfigYAML(cfg)

	if err := os.WriteFile(filepath, []byte(data), 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

func generateConfigYAML(cfg *Config) string {
	return fmt.Sprintf(`# MikaBooM Resource Monitor Configuration File
# 系统资源监控与调整工具 - 配置文件

# CPU占用率阈值 (0-100)
# 当其他程序CPU占用低于此阈值时，本程序会启动CPU计算负载
cpu_threshold: %d

# 内存占用率阈值 (0-100)
# 当其他程序内存占用低于此阈值时，本程序会启动内存计算负载
memory_threshold: %d

# 是否开机自启动
# true: 启用自启动, false: 禁用自启动
auto_start: %t

# 是否显示窗口
# true: 显示命令行窗口（前台运行）, false: 隐藏窗口（后台运行）
show_window: %t

# 监控更新间隔（秒）
# 建议值: 1-10 秒
update_interval: %d

# 通知设置
notification:
  # 是否启用系统通知
  enabled: %t
  # 通知冷却时间（秒）
  # 避免频繁通知，相同类型的通知在冷却时间内只会显示一次
  cooldown: %d

# 更新检查设置
update_check:
  # 是否启用更新检查
  enabled: %t
  # 是否在启动时检查更新
  check_on_startup: %t
  # 是否自动下载更新（暂未实现，需要手动运行 -update）
  auto_download: %t
  # 是否静默检查（不显示"已是最新版本"的提示）
  silent_check: %t
`,
		cfg.CPUThreshold,
		cfg.MemoryThreshold,
		cfg.AutoStart,
		cfg.ShowWindow,
		cfg.UpdateInterval,
		cfg.Notification.Enabled,
		cfg.Notification.Cooldown,
		cfg.UpdateCheck.Enabled,
		cfg.UpdateCheck.CheckOnStartup,
		cfg.UpdateCheck.AutoDownload,
		cfg.UpdateCheck.SilentCheck,
	)
}

func FindConfigFile(specifiedPath string) (string, error) {
	if specifiedPath != "" {
		if filepath.IsAbs(specifiedPath) {
			return specifiedPath, nil
		}

		exePath, err := os.Executable()
		if err == nil {
			exeDir := filepath.Dir(exePath)
			absPath := filepath.Join(exeDir, specifiedPath)
			return absPath, nil
		}

		absPath, _ := filepath.Abs(specifiedPath)
		return absPath, nil
	}

	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		configPath := filepath.Join(exeDir, "config.yaml")
		return configPath, nil
	}

	workDir, err := os.Getwd()
	if err == nil {
		configPath := filepath.Join(workDir, "config.yaml")
		return configPath, nil
	}

	return "config.yaml", nil
}

func ValidateConfig(cfg *Config) error {
	if cfg.CPUThreshold < 0 || cfg.CPUThreshold > 100 {
		return fmt.Errorf("CPU阈值必须在 0-100 之间，当前值: %d", cfg.CPUThreshold)
	}

	if cfg.MemoryThreshold < 0 || cfg.MemoryThreshold > 100 {
		return fmt.Errorf("内存阈值必须在 0-100 之间，当前值: %d", cfg.MemoryThreshold)
	}

	if cfg.UpdateInterval < 1 {
		return fmt.Errorf("更新间隔必须大于 0，当前值: %d", cfg.UpdateInterval)
	}

	if cfg.Notification.Cooldown < 0 {
		return fmt.Errorf("通知冷却时间不能为负数，当前值: %d", cfg.Notification.Cooldown)
	}

	return nil
}

func ReloadConfig(filepath string) (*Config, error) {
	return LoadConfig(filepath)
}

func UpdateConfig(filepath string, updateFn func(*Config)) error {
	cfg, err := LoadConfig(filepath)
	if err != nil {
		return err
	}

	updateFn(cfg)

	if err := ValidateConfig(cfg); err != nil {
		return err
	}

	return SaveConfig(filepath, cfg)
}

func ConfigExists(filepath string) bool {
	_, err := os.Stat(filepath)
	return err == nil
}

func GetConfigInfo(filepath string) (map[string]interface{}, error) {
	info := make(map[string]interface{})

	stat, err := os.Stat(filepath)
	if err != nil {
		return nil, err
	}

	info["path"] = filepath
	info["size"] = stat.Size()
	info["modified"] = stat.ModTime()
	info["exists"] = true

	return info, nil
}