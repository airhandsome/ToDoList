package config

import (
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	App      AppConfig      `yaml:"app"`
	Pomodoro PomodoroConfig `yaml:"pomodoro"`
	Database DatabaseConfig `yaml:"database"`
	Theme    ThemeConfig    `yaml:"theme"`
}

type AppConfig struct {
	Name         string `yaml:"name"`
	Version      string `yaml:"version"`
	WindowWidth  int    `yaml:"window_width"`
	WindowHeight int    `yaml:"window_height"`
}

type PomodoroConfig struct {
	WorkDuration      time.Duration `yaml:"work_duration"`
	ShortBreak        time.Duration `yaml:"short_break"`
	LongBreak         time.Duration `yaml:"long_break"`
	LongBreakAfter    int           `yaml:"long_break_after"`
	AutoStartBreak    bool          `yaml:"auto_start_break"`
	AutoStartPomodoro bool          `yaml:"auto_start_pomodoro"`
	NotificationSound bool          `yaml:"notification_sound"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type ThemeConfig struct {
	DarkMode bool   `yaml:"dark_mode"`
	FontSize int    `yaml:"font_size"`
	Language string `yaml:"language"`
}

// 默认配置
func DefaultConfig() *Config {
	return &Config{
		App: AppConfig{
			Name:         "番茄钟 + 待办事项",
			Version:      "1.0.0",
			WindowWidth:  800,
			WindowHeight: 600,
		},
		Pomodoro: PomodoroConfig{
			WorkDuration:      25 * time.Minute,
			ShortBreak:        5 * time.Minute,
			LongBreak:         15 * time.Minute,
			LongBreakAfter:    4,
			AutoStartBreak:    false,
			AutoStartPomodoro: false,
			NotificationSound: true,
		},
		Database: DatabaseConfig{
			Path: "pomodoro.db",
		},
		Theme: ThemeConfig{
			DarkMode: false,
			FontSize: 14,
			Language: "zh-CN",
		},
	}
}

type Manager struct {
	config     *Config
	configPath string
}

func NewManager() (*Manager, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(configDir, "config.yaml")
	manager := &Manager{
		configPath: configPath,
	}

	// 加载或创建配置
	if err := manager.loadConfig(); err != nil {
		manager.config = DefaultConfig()
		if err := manager.SaveConfig(); err != nil {
			return nil, err
		}
	}

	return manager, nil
}

func (m *Manager) loadConfig() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return err
	}

	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return err
	}

	m.config = config
	return nil
}

func (m *Manager) SaveConfig() error {
	data, err := yaml.Marshal(m.config)
	if err != nil {
		return err
	}

	// 确保配置目录存在
	configDir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	return os.WriteFile(m.configPath, data, 0644)
}

func (m *Manager) GetConfig() *Config {
	return m.config
}

// 获取配置文件目录
func getConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// 在用户目录下创建应用配置目录
	configDir := filepath.Join(homeDir, ".pomodoro-todo")
	return configDir, nil
}

// 更新配置的便捷方法
func (m *Manager) UpdatePomodoroConfig(config PomodoroConfig) error {
	m.config.Pomodoro = config
	return m.SaveConfig()
}

func (m *Manager) UpdateThemeConfig(config ThemeConfig) error {
	m.config.Theme = config
	return m.SaveConfig()
}

// 监听配置变化
type ConfigChangeCallback func(*Config)

func (m *Manager) WatchConfig(callback ConfigChangeCallback) {
	// TODO: 实现配置文件监听
	// 可以使用 fsnotify 包来监听配置文件变化
}
