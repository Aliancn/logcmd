package modes

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aliancn/logcmd/internal/ui/common"
)

// LogViewMode 日志查看模式
//
// 核心功能：
//   - 显示完整的日志文件内容
//   - 高亮显示匹配行
//   - 支持滚动浏览
//   - 支持搜索定位
type LogViewMode struct {
	// 数据
	filePath    string
	lineNum     int    // 目标行号
	searchQuery string // 搜索关键词（用于高亮）

	// UI 组件
	viewport viewport.Model

	// 状态
	content string
	loaded  bool
	error   error

	// 布局
	width  int
	height int

	// 样式
	theme  common.Theme
	styles common.Styles
}

// NewLogViewMode 创建日志查看模式
func NewLogViewMode(theme common.Theme, styles common.Styles) *LogViewMode {
	vp := viewport.New(0, 0)

	return &LogViewMode{
		viewport: vp,
		theme:    theme,
		styles:   styles,
	}
}

// Name 实现 Mode 接口
func (m *LogViewMode) Name() string {
	return "logview"
}

// Activate 实现 Mode 接口
func (m *LogViewMode) Activate() tea.Cmd {
	// 激活时加载文件内容
	if !m.loaded && m.filePath != "" {
		return m.loadFileCmd()
	}
	return nil
}

// Deactivate 实现 Mode 接口
func (m *LogViewMode) Deactivate() tea.Cmd {
	// 清理状态
	m.content = ""
	m.loaded = false
	m.error = nil
	return nil
}

// Update 实现 Mode 接口
func (m *LogViewMode) Update(msg tea.Msg) (Mode, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// viewport高度 = 总高度 - 状态栏(3行) - 底部提示(2行)
		vpHeight := msg.Height - 5
		if vpHeight < 5 {
			vpHeight = 5
		}
		m.viewport.Width = msg.Width - 2
		m.viewport.Height = vpHeight

		return m, nil

	case fileLoadedMsg:
		m.content = msg.content
		m.loaded = true
		m.error = nil

		// 设置viewport内容
		m.viewport.SetContent(m.renderContent())

		// 跳转到目标行
		if m.lineNum > 0 {
			m.gotoLine(m.lineNum)
		}

		return m, nil

	case fileLoadFailedMsg:
		m.error = msg.err
		m.loaded = true
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			// 返回搜索模式
			return m, func() tea.Msg {
				return SwitchModeMsg{ModeName: "search"}
			}
		}
	}

	// 更新viewport（处理滚动）
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View 实现 Mode 接口
func (m *LogViewMode) View() string {
	if m.width == 0 {
		return "初始化中..."
	}

	// 状态栏
	statusBar := m.renderStatusBar()

	// 内容区域
	var content string
	if !m.loaded {
		content = m.renderLoading()
	} else if m.error != nil {
		content = m.renderError()
	} else {
		content = m.viewport.View()
	}

	// 底部提示
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left,
		statusBar,
		content,
		footer,
	)
}

// HandleKey 实现 Mode 接口
func (m *LogViewMode) HandleKey(key string) (bool, tea.Cmd) {
	// 所有快捷键在Update中处理
	return false, nil
}

// SetFile 设置要查看的文件
func (m *LogViewMode) SetFile(filePath string, lineNum int, searchQuery string) {
	m.filePath = filePath
	m.lineNum = lineNum
	m.searchQuery = searchQuery
	m.loaded = false
	m.error = nil
}

// loadFileCmd 加载文件内容
func (m *LogViewMode) loadFileCmd() tea.Cmd {
	return func() tea.Msg {
		data, err := os.ReadFile(m.filePath)
		if err != nil {
			return fileLoadFailedMsg{err: fmt.Errorf("读取文件失败: %w", err)}
		}

		return fileLoadedMsg{content: string(data)}
	}
}

// renderContent 渲染文件内容（带行号和高亮）
func (m *LogViewMode) renderContent() string {
	if m.content == "" {
		return "(空文件)"
	}

	lines := strings.Split(m.content, "\n")
	var result strings.Builder

	// 高亮样式
	highlightStyle := lipgloss.NewStyle().
		Background(m.theme.Warning).
		Foreground(m.theme.Background)

	matchLineStyle := lipgloss.NewStyle().
		Background(m.theme.StatusBar).
		Foreground(m.theme.Foreground)

	lineNumStyle := lipgloss.NewStyle().
		Foreground(m.theme.TextMuted).
		Width(6).
		Align(lipgloss.Right)

	for i, line := range lines {
		lineNum := i + 1
		isMatchLine := lineNum == m.lineNum

		// 行号
		lineNumText := lineNumStyle.Render(fmt.Sprintf("%d ", lineNum))

		// 高亮搜索关键词
		displayLine := line
		if m.searchQuery != "" && strings.Contains(strings.ToLower(line), strings.ToLower(m.searchQuery)) {
			// 简单的关键词高亮（不区分大小写）
			displayLine = highlightKeyword(line, m.searchQuery, highlightStyle)
		}

		// 如果是匹配行，整行使用不同背景
		if isMatchLine {
			displayLine = matchLineStyle.Render("> " + displayLine)
			result.WriteString(lineNumText + displayLine + "\n")
		} else {
			result.WriteString(lineNumText + "  " + displayLine + "\n")
		}
	}

	return result.String()
}

// highlightKeyword 高亮关键词
func highlightKeyword(text, keyword string, style lipgloss.Style) string {
	if keyword == "" {
		return text
	}

	lowerText := strings.ToLower(text)
	lowerKeyword := strings.ToLower(keyword)

	var result strings.Builder
	lastIndex := 0

	for {
		index := strings.Index(lowerText[lastIndex:], lowerKeyword)
		if index == -1 {
			result.WriteString(text[lastIndex:])
			break
		}

		actualIndex := lastIndex + index
		result.WriteString(text[lastIndex:actualIndex])
		result.WriteString(style.Render(text[actualIndex : actualIndex+len(keyword)]))
		lastIndex = actualIndex + len(keyword)
	}

	return result.String()
}

// gotoLine 跳转到指定行
func (m *LogViewMode) gotoLine(lineNum int) {
	if lineNum <= 0 {
		return
	}

	lines := strings.Split(m.content, "\n")
	if lineNum > len(lines) {
		lineNum = len(lines)
	}

	// 计算目标行的偏移（每行包含行号）
	// 让目标行显示在viewport的中间位置
	targetOffset := lineNum - m.viewport.Height/2
	if targetOffset < 0 {
		targetOffset = 0
	}

	m.viewport.SetYOffset(targetOffset)
}

// renderStatusBar 渲染状态栏
func (m *LogViewMode) renderStatusBar() string {
	fileName := m.filePath
	if len(fileName) > 60 {
		fileName = "..." + fileName[len(fileName)-57:]
	}

	status := fmt.Sprintf("[日志查看] %s", fileName)
	if m.lineNum > 0 {
		status += fmt.Sprintf(" · 行 %d", m.lineNum)
	}

	statusStyle := lipgloss.NewStyle().
		Foreground(m.theme.Foreground).
		Background(m.theme.StatusBar).
		Padding(0, 1).
		Width(m.width)

	return statusStyle.Render(status)
}

// renderFooter 渲染底部提示
func (m *LogViewMode) renderFooter() string {
	hints := "↑↓ 滚动 · PgUp/PgDn 翻页 · q/Esc 返回搜索"

	footerStyle := lipgloss.NewStyle().
		Foreground(m.theme.TextMuted).
		Padding(0, 1)

	return footerStyle.Render(hints)
}

// renderLoading 渲染加载状态
func (m *LogViewMode) renderLoading() string {
	if m.height == 0 {
		return "加载中..."
	}

	padding := (m.height - 5) / 2
	if padding < 0 {
		padding = 0
	}

	var view string
	for i := 0; i < padding; i++ {
		view += "\n"
	}

	loadingStyle := lipgloss.NewStyle().
		Foreground(m.theme.Primary).
		Bold(true)

	view += loadingStyle.Render("⏳ 正在加载日志文件...")
	return view
}

// renderError 渲染错误信息
func (m *LogViewMode) renderError() string {
	if m.height == 0 {
		return fmt.Sprintf("错误: %v", m.error)
	}

	padding := (m.height - 5) / 2
	if padding < 0 {
		padding = 0
	}

	var view string
	for i := 0; i < padding; i++ {
		view += "\n"
	}

	errorStyle := lipgloss.NewStyle().
		Foreground(m.theme.Error).
		Bold(true)

	view += errorStyle.Render(fmt.Sprintf("❌ %v", m.error))
	return view
}

// 消息类型

type fileLoadedMsg struct {
	content string
}

type fileLoadFailedMsg struct {
	err error
}
