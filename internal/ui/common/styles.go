package common

import "github.com/charmbracelet/lipgloss"

// Styles 统一管理UI样式
type Styles struct {
	// 边框样式
	ActiveBorder   lipgloss.Style // 激活状态的圆角边框
	InactiveBorder lipgloss.Style // 非激活状态的圆角边框
	NoBorder       lipgloss.Style // 无边框样式
	Frame          lipgloss.Style // 框架样式（向后兼容）

	// 文本样式
	Title         lipgloss.Style // 标题文本
	Subtitle      lipgloss.Style // 副标题文本
	Success       lipgloss.Style // 成功状态文本
	Warning       lipgloss.Style // 警告状态文本
	Error         lipgloss.Style // 错误状态文本
	Muted         lipgloss.Style // 次要文本
	Highlight     lipgloss.Style // 高亮文本
	Normal        lipgloss.Style // 普通文本

	// UI元素样式
	StatusBar     lipgloss.Style // 状态栏
	TabActive     lipgloss.Style // 激活的Tab标签
	TabInactive   lipgloss.Style // 非激活的Tab标签
	AppContainer  lipgloss.Style // 应用容器

	// 进度条和图表
	ProgressFilled lipgloss.Style // 进度条填充部分
	ProgressEmpty  lipgloss.Style // 进度条空白部分

	// 列表项样式
	ListItem         lipgloss.Style // 普通列表项
	ListItemSelected lipgloss.Style // 选中的列表项
}

// NewStyles 基于Theme创建样式集
func NewStyles(theme Theme) Styles {
	return Styles{
		// 圆角边框样式
		ActiveBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.BorderActive).
			Padding(1),

		InactiveBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Border).
			Padding(1),

		NoBorder: lipgloss.NewStyle().
			Padding(1),

		Frame: lipgloss.NewStyle().
			Margin(1, 2),

		// 文本样式
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Primary).
			PaddingBottom(1),

		Subtitle: lipgloss.NewStyle().
			Foreground(theme.Primary).
			PaddingBottom(1),

		Success: lipgloss.NewStyle().
			Foreground(theme.Success).
			Bold(true),

		Warning: lipgloss.NewStyle().
			Foreground(theme.Warning).
			Bold(true),

		Error: lipgloss.NewStyle().
			Foreground(theme.Error).
			Bold(true),

		Muted: lipgloss.NewStyle().
			Foreground(theme.TextMuted),

		Highlight: lipgloss.NewStyle().
			Foreground(theme.TextHighlight).
			Bold(true),

		Normal: lipgloss.NewStyle().
			Foreground(theme.Foreground),

		// UI元素样式
		StatusBar: lipgloss.NewStyle().
			Foreground(theme.Foreground).
			Background(theme.StatusBar).
			Padding(0, 1),

		TabActive: lipgloss.NewStyle().
			Foreground(theme.Foreground).
			Background(theme.TabActive).
			Bold(true).
			Padding(0, 2),

		TabInactive: lipgloss.NewStyle().
			Foreground(theme.TabInactive).
			Padding(0, 2),

		AppContainer: lipgloss.NewStyle(),

		// 进度条样式
		ProgressFilled: lipgloss.NewStyle().
			Foreground(theme.Success),

		ProgressEmpty: lipgloss.NewStyle().
			Foreground(theme.TextMuted),

		// 列表项样式
		ListItem: lipgloss.NewStyle().
			Foreground(theme.Foreground),

		ListItemSelected: lipgloss.NewStyle().
			Foreground(theme.Foreground).
			Background(theme.Primary).
			Bold(true),
	}
}

// DefaultStyles 返回基于默认主题的样式集
func DefaultStyles() Styles {
	return NewStyles(DefaultTheme())
}

// 辅助函数：创建进度条
func RenderProgressBar(filled, total int, width int, styles Styles) string {
	if total == 0 {
		return ""
	}

	filledWidth := (filled * width) / total
	if filledWidth > width {
		filledWidth = width
	}

	filledBar := styles.ProgressFilled.Render(lipgloss.NewStyle().Width(filledWidth).Render("█"))
	emptyWidth := width - filledWidth
	emptyBar := ""
	if emptyWidth > 0 {
		emptyBar = styles.ProgressEmpty.Render(lipgloss.NewStyle().Width(emptyWidth).Render("░"))
	}

	return filledBar + emptyBar
}
