package template

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ElementType 命名元素类型
type ElementType string

const (
	ElementTypeCommand ElementType = "command" // 命令内容
	ElementTypeTime    ElementType = "time"    // 时间
	ElementTypeProject ElementType = "project" // 项目名称
	ElementTypeCustom  ElementType = "custom"  // 自定义内容
)

// NameElement 命名元素
type NameElement struct {
	Type   ElementType       `json:"type"`   // 元素类型
	Config map[string]string `json:"config"` // 元素配置
}

// LogNameTemplate 日志命名模板
type LogNameTemplate struct {
	Elements  []NameElement `json:"elements"`  // 命名元素列表（按顺序）
	Separator string        `json:"separator"` // 元素分隔符
}

// DefaultTemplate 返回默认模板
func DefaultTemplate() *LogNameTemplate {
	return &LogNameTemplate{
		Separator: "_",
		Elements: []NameElement{
			{
				Type: ElementTypeTime,
			},
		},
	}
}

// GetConfigPath 获取模板配置文件路径
func GetConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户目录失败: %w", err)
	}

	// 创建统一的配置目录
	logcmdDir := filepath.Join(home, ".logcmd")
	configDir := filepath.Join(logcmdDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("创建配置目录失败: %w", err)
	}

	return filepath.Join(configDir, "template.json"), nil
}

// Load 加载模板配置
func Load() (*LogNameTemplate, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// 如果配置文件不存在，返回默认模板
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultTemplate(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var template LogNameTemplate
	if err := json.Unmarshal(data, &template); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return &template, nil
}

// Save 保存模板配置
func (t *LogNameTemplate) Save() error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// GenerateLogName 根据模板生成日志文件名
func (t *LogNameTemplate) GenerateLogName(command string, args []string, projectName string, timezone *time.Location, timeFormat string) string {
	tz := timezone
	if tz == nil {
		tz = time.Local
	}
	now := time.Now().In(tz)

	var parts []string

	for _, element := range t.Elements {
		var part string

		switch element.Type {
		case ElementTypeCommand:
			// 使用命令名称
			part = sanitizeFilename(command)
		case ElementTypeTime:
			// 使用时间
			part = now.Format(timeFormat)
		case ElementTypeProject:
			// 使用项目名称
			part = sanitizeFilename(projectName)
		case ElementTypeCustom:
			// 使用自定义文本
			part = sanitizeFilename(element.Config["text"])
		}

		if part != "" {
			parts = append(parts, part)
		}
	}

	// 如果没有任何元素，使用默认命名
	if len(parts) == 0 {
		return now.Format("log_20060102_150405.log")
	}

	return strings.Join(parts, t.Separator) + ".log"
}

// sanitizeFilename 清理文件名，移除不安全字符
func sanitizeFilename(name string) string {
	// 1. 替换不安全字符为下划线
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
		"\x00", "", // Null byte
	)
	clean := replacer.Replace(name)

	// 2. 移除头部和尾部的点和空格 (Windows 不喜欢)
	clean = strings.Trim(clean, ". ")

	// 3. 检查 Windows 保留字
	upper := strings.ToUpper(clean)
	reserved := map[string]bool{
		"CON": true, "PRN": true, "AUX": true, "NUL": true,
		"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true, "COM6": true, "COM7": true, "COM8": true, "COM9": true,
		"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true, "LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
	}
	// 如果是保留字，添加下划线后缀
	if reserved[upper] {
		clean = clean + "_"
	}

	// 4. 限制长度 (255 max, safe limit 200)
	if len(clean) > 200 {
		clean = clean[:200]
	}

	if clean == "" {
		return "unknown"
	}

	return clean
}

// GetProjectName 从.logcmd目录获取项目名称
func GetProjectName(logDir string) string {
	// 获取.logcmd目录的绝对路径
	absPath, err := filepath.Abs(logDir)
	if err != nil {
		return "unknown"
	}

	// 获取父目录名称
	parentDir := filepath.Dir(absPath)
	projectName := filepath.Base(parentDir)

	if projectName == "" || projectName == "." || projectName == "/" {
		return "unknown"
	}

	return projectName
}
