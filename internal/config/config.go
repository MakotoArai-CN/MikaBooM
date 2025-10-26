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
	EnableWorker    bool               `yaml:"-"`
}

type NotificationConfig struct {
	Enabled  bool `yaml:"enabled"`
	Cooldown int  `yaml:"cooldown"`
}

// GetDefaultConfig 返回默认配置
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
		EnableWorker: true,
	}
}

// LoadConfig 加载配置文件
// 如果文件不存在，会创建默认配置文件
func LoadConfig(filepath string) (*Config, error) {
	cfg := GetDefaultConfig()

	// 检查文件是否存在
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		// 文件不存在，创建默认配置文件
		if err := SaveConfig(filepath, cfg); err != nil {
			return cfg, fmt.Errorf("创建默认配置文件失败: %w", err)
		}
		return cfg, nil
	}

	// 文件存在，读取配置
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return cfg, nil
}

// SaveConfig 保存配置文件
func SaveConfig(filepath string, cfg *Config) error {
	// 确保目录存在
	dir := filepath[:len(filepath)-len(filepath[len(filepath)-1:])]
	if dir != "" {
		dir = filepath[:len(filepath)-len(filepath[len(filepath)-1:])]
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建配置目录失败: %w", err)
		}
	}

	// 生成带注释的 YAML
	data := generateConfigYAML(cfg)

	if err := os.WriteFile(filepath, []byte(data), 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// generateConfigYAML 生成带注释的配置文件内容
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
`,
		cfg.CPUThreshold,
		cfg.MemoryThreshold,
		cfg.AutoStart,
		cfg.ShowWindow,
		cfg.UpdateInterval,
		cfg.Notification.Enabled,
		cfg.Notification.Cooldown,
	)
}

// FindConfigFile 自动查找配置文件
// 查找顺序：
// 1. 指定的配置文件路径（如果提供）
// 2. 可执行文件同级目录下的 config.yaml
// 3. 当前工作目录下的 config.yaml
func FindConfigFile(specifiedPath string) (string, error) {
	// 如果指定了配置文件，直接使用
	if specifiedPath != "" {
		// 如果是绝对路径，直接返回
		if filepath.IsAbs(specifiedPath) {
			return specifiedPath, nil
		}

		// 相对路径，先尝试基于可执行文件目录
		exePath, err := os.Executable()
		if err == nil {
			exeDir := filepath.Dir(exePath)
			absPath := filepath.Join(exeDir, specifiedPath)
			return absPath, nil
		}

		// 如果获取可执行文件路径失败，返回相对路径
		absPath, _ := filepath.Abs(specifiedPath)
		return absPath, nil
	}

	// 没有指定配置文件，按顺序查找

	// 1. 可执行文件同级目录
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		configPath := filepath.Join(exeDir, "config.yaml")
		return configPath, nil
	}

	// 2. 当前工作目录
	workDir, err := os.Getwd()
	if err == nil {
		configPath := filepath.Join(workDir, "config.yaml")
		return configPath, nil
	}

	// 3. 默认使用相对路径
	return "config.yaml", nil
}

// ValidateConfig 验证配置的有效性
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

// ReloadConfig 重新加载配置文件
func ReloadConfig(filepath string) (*Config, error) {
	return LoadConfig(filepath)
}

// UpdateConfig 更新配置并保存
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

// ConfigExists 检查配置文件是否存在
func ConfigExists(filepath string) bool {
	_, err := os.Stat(filepath)
	return err == nil
}

// GetConfigInfo 获取配置文件信息
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