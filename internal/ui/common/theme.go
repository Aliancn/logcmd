package common

import "github.com/charmbracelet/lipgloss"

// Theme 定义UI配色方案
type Theme struct {
	// 基础色
	Background lipgloss.Color // 背景色
	Foreground lipgloss.Color // 前景色（主文本）

	// 语义色
	Primary lipgloss.Color // 主强调色
	Success lipgloss.Color // 成功/运行状态
	Warning lipgloss.Color // 警告状态
	Error   lipgloss.Color // 错误/失败状态

	// UI元素色
	Border       lipgloss.Color // 普通边框
	BorderActive lipgloss.Color // 激活边框
	StatusBar    lipgloss.Color // 状态栏背景
	TabActive    lipgloss.Color // 激活Tab
	TabInactive  lipgloss.Color // 非激活Tab

	// 文本色
	TextMuted     lipgloss.Color // 次要文本
	TextHighlight lipgloss.Color // 高亮文本
}

// LightTheme 返回明亮色主题（默认）
func LightTheme() Theme {
	return Theme{
		Background:    lipgloss.Color("#ffffff"),
		Foreground:    lipgloss.Color("#000000"), // 纯黑文本
		Primary:       lipgloss.Color("#0056b3"), // 深蓝强调色
		Success:       lipgloss.Color("#28a745"), // 绿色
		Warning:       lipgloss.Color("#d39e00"), // 深黄色（为了在白底可见）
		Error:         lipgloss.Color("#dc3545"), // 红色
		Border:        lipgloss.Color("#dee2e6"), // 浅灰边框
		BorderActive:  lipgloss.Color("#0056b3"), // 激活边框同主色
		StatusBar:     lipgloss.Color("#f8f9fa"), // 浅灰背景
		TabActive:     lipgloss.Color("#e9ecef"), // 激活Tab背景
		TabInactive:   lipgloss.Color("#dee2e6"), // 非激活Tab背景
		TextMuted:     lipgloss.Color("#6c757d"), // 灰色文本
		TextHighlight: lipgloss.Color("#e83e8c"), // 高亮色
	}
}

// DraculaTheme 返回Dracula配色方案 (保留备用)
func DraculaTheme() Theme {
	return Theme{
		Background:    lipgloss.Color("#282a36"),
		Foreground:    lipgloss.Color("#f8f8f2"),
		Primary:       lipgloss.Color("#bd93f9"), // 霓虹紫
		Success:       lipgloss.Color("#50fa7b"), // 春绿色
		Warning:       lipgloss.Color("#ffb86c"), // 橙色
		Error:         lipgloss.Color("#ff5555"), // 鲜红色
		Border:        lipgloss.Color("#6272a4"), // 灰蓝
		BorderActive:  lipgloss.Color("#bd93f9"), // 紫色
		StatusBar:     lipgloss.Color("#44475a"), // 深灰
		TabActive:     lipgloss.Color("#bd93f9"),
		TabInactive:   lipgloss.Color("#6272a4"),
		TextMuted:     lipgloss.Color("#6272a4"),
		TextHighlight: lipgloss.Color("#f1fa8c"), // 黄色
	}
}

// DefaultTheme 返回默认主题
func DefaultTheme() Theme {
	return LightTheme()
}