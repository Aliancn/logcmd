package panel

import (
	"strings"

	"github.com/aliancn/logcmd/internal/ui/common"
	"github.com/charmbracelet/lipgloss"
)

// Panel 统一的布局容器，负责边框、padding和尺寸计算
type Panel struct {
	// 配置
	title       string
	showBorder  bool
	showPadding bool
	theme       common.Theme
	styles      common.Styles

	// 尺寸
	width  int
	height int

	// 计算后的内容区域尺寸
	contentWidth  int
	contentHeight int

	// 可选的header和footer
	header string
	footer string
}

// Config Panel配置选项
type Config struct {
	Title       string
	ShowBorder  bool
	ShowPadding bool
	Header      string
	Footer      string
}

// New 创建新的Panel
func New(theme common.Theme, styles common.Styles, config Config) *Panel {
	return &Panel{
		title:       config.Title,
		showBorder:  config.ShowBorder,
		showPadding: config.ShowPadding,
		header:      config.Header,
		footer:      config.Footer,
		theme:       theme,
		styles:      styles,
	}
}

// NewDefault 创建带默认配置的Panel（有边框和padding）
func NewDefault(title string, theme common.Theme, styles common.Styles) *Panel {
	return New(theme, styles, Config{
		Title:       title,
		ShowBorder:  true,
		ShowPadding: true,
	})
}

// SetSize 设置Panel的总尺寸，自动计算内容区域尺寸
func (p *Panel) SetSize(width, height int) {
	p.width = width
	p.height = height
	p.calculateContentSize()
}

// SetHeader 设置header内容
func (p *Panel) SetHeader(header string) {
	p.header = header
	p.calculateContentSize()
}

// SetFooter 设置footer内容
func (p *Panel) SetFooter(footer string) {
	p.footer = footer
	p.calculateContentSize()
}

// GetContentSize 返回内容区域的精确尺寸
func (p *Panel) GetContentSize() (width, height int) {
	return p.contentWidth, p.contentHeight
}

// calculateContentSize 计算内容区域的可用尺寸
func (p *Panel) calculateContentSize() {
	// 从总尺寸开始
	contentW := p.width
	contentH := p.height

	// 减去边框占用（如果有）
	if p.showBorder {
		contentW -= 2 // 左右边框各1
		contentH -= 2 // 上下边框各1
	}

	// 减去padding占用（如果有）
	if p.showPadding {
		contentW -= 2 // 左右padding各1
		// contentH -= 0 // 上下padding为0 (Render中使用 Padding(0, 1))
	}

	// 减去header占用（如果有）
	if p.header != "" {
		headerLines := blockHeight(p.header)
		contentH -= headerLines
		contentH -= 1 // header和内容之间的分隔行
	}

	// 减去footer占用（如果有）
	if p.footer != "" {
		footerLines := blockHeight(p.footer)
		contentH -= footerLines
	}

	// 确保最小尺寸
	if contentW < 0 {
		contentW = 0
	}
	if contentH < 0 {
		contentH = 0
	}

	p.contentWidth = contentW
	p.contentHeight = contentH
}

// Render 渲染Panel，将内容包裹在边框和装饰中
func (p *Panel) Render(content string) string {
	if p.width == 0 || p.height == 0 {
		return ""
	}

	// 构建最终内容
	var finalContent string

	// 添加header（如果有）
	if p.header != "" {
		finalContent = p.header + "\n"
	}

	// 添加主内容
	finalContent += content

	// 计算并填充空白行，将Footer推到底部
	// 可用总高度
	availableHeight := p.height
	if p.showBorder {
		availableHeight -= 2
	}
	// content height
	currentLines := lipgloss.Height(finalContent)

	// footer height
	footerHeight := 0
	if p.footer != "" {
		footerHeight = blockHeight(p.footer)
	}

	// 需要填充的行数
	neededFill := availableHeight - currentLines - footerHeight
	if neededFill > 0 {
		finalContent += strings.Repeat("\n", neededFill)
	}

	// 添加footer（如果有）
	if p.footer != "" {
		// 确保footer占满宽度且背景一致
		footerStyle := p.styles.StatusBar.Copy().Width(p.contentWidth)
		finalContent += "\n" + footerStyle.Render(p.footer)
	}

	// 计算内部可用高度 (Total - Border)
	innerHeight := p.height
	if p.showBorder {
		innerHeight -= 2
	}
	// Padding vertical is 0, so no need to subtract

	// 强制截断内容以防止溢出
	if innerHeight > 0 {
		lines := strings.Split(finalContent, "\n")
		if len(lines) > innerHeight {
			lines = lines[:innerHeight]
			finalContent = strings.Join(lines, "\n")
		}
	}

	// 应用边框和padding样式
	style := lipgloss.NewStyle()

	innerWidth := p.width
	if p.showBorder {
		style = style.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(p.theme.BorderActive)
		innerWidth -= 2
	}

	if p.showPadding {
		style = style.Padding(0, 1) // 上下0，左右1
	}

	// 强制高度，确保布局对齐
	if innerHeight > 0 {
		style = style.Height(innerHeight)
	}

	// 强制宽度，确保布局占满
	if innerWidth > 0 {
		style = style.Width(innerWidth)
	}

	// 渲染并返回
	return style.Render(finalContent)
}

// RenderEmpty 渲染空内容时的占位符
func (p *Panel) RenderEmpty(message string) string {
	emptyStyle := lipgloss.NewStyle().
		Foreground(p.theme.TextMuted).
		Width(p.contentWidth).
		Height(p.contentHeight).
		Align(lipgloss.Center, lipgloss.Center)

	return p.Render(emptyStyle.Render(message))
}

// blockHeight 计算渲染后的文本高度（考虑padding/样式换行）
func blockHeight(text string) int {
	if text == "" {
		return 0
	}
	return lipgloss.Height(text)
}

// Width 返回Panel的总宽度
func (p *Panel) Width() int {
	return p.width
}

// Height 返回Panel的总高度
func (p *Panel) Height() int {
	return p.height
}
