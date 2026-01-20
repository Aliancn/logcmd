package config

// PredefinedTimeFormats 定义允许的时间格式映射
// key 为简短的标识符或格式本身，value 为实际的 Go 时间格式字符串
var PredefinedTimeFormats = map[string]string{
	"compact":  "20060102_150405",     // 紧凑格式 (默认)
	"standard": "2006-01-02_15-04-05", // 标准格式
	"simple":   "20060102-150405",     // 简单格式
	"dateonly": "20060102",            // 仅日期
}

// GetTimeFormatDescriptions 返回所有可用格式的描述列表
func GetTimeFormatDescriptions() []string {
	return []string{
		"compact  : 20060102_150405     (默认, 如: 20251230_143000)",
		"standard : 2006-01-02_15-04-05 (如: 2025-12-30_14-30-00)",
		"simple   : 20060102-150405     (如: 20251230-143000)",
		"dateonly : 20060102            (如: 20251230)",
	}
}

// IsValidTimeFormat 检查是否为有效的预定义格式标识符
func IsValidTimeFormat(name string) bool {
	_, ok := PredefinedTimeFormats[name]
	return ok
}

// GetTimeFormat 根据标识符获取实际格式字符串
func GetTimeFormat(name string) string {
	if format, ok := PredefinedTimeFormats[name]; ok {
		return format
	}
	return PredefinedTimeFormats["compact"] // 默认返回 compact
}
