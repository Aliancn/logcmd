package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// PersistentConfig 定义可持久化的配置项
type PersistentConfig struct {
	BufferSize   int    `json:"buffer_size,omitempty"`   // 缓冲区大小
	AutoCompress *bool  `json:"auto_compress,omitempty"` // 是否自动压缩
	TimeFormat   string `json:"time_format,omitempty"`   // 时间格式
}

// DefaultPersistentConfig 返回默认持久化配置
func DefaultPersistentConfig() PersistentConfig {
	return PersistentConfig{
		BufferSize:   8192,
		AutoCompress: nil,
		TimeFormat:   "20060102_150405",
	}
}

// GetGlobalConfigPath 获取全局配置文件路径 (~/.logcmd/config.json)
func GetGlobalConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户目录失败: %w", err)
	}

	configDir := filepath.Join(home, ".logcmd")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("创建配置目录失败: %w", err)
	}

	return filepath.Join(configDir, "config.json"), nil
}

// GetLocalConfigPath 获取局部配置文件路径 (当前项目 .logcmd/config.json)
// searchDir 是开始查找的目录，通常是当前工作目录
func GetLocalConfigPath(searchDir string) (string, error) {
	// 复用 findLogDir 的逻辑找到 .logcmd 目录
	logCmdDir := findLogDirFrom(searchDir)
	if logCmdDir == "" {
		// 如果没找到现有的，通常我们假设它会在当前目录的 .logcmd 下（如果用户想创建的话）
		// 但对于读取来说，如果没有就不读
		return "", os.ErrNotExist
	}

	return filepath.Join(logCmdDir, "config.json"), nil
}

// LoadConfigFile 读取指定路径的配置文件
func LoadConfigFile(path string) (*PersistentConfig, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil // 文件不存在不是错误，返回 nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg PersistentConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return &cfg, nil
}

// SaveConfigFile 保存配置到指定路径
func SaveConfigFile(path string, cfg PersistentConfig) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}
