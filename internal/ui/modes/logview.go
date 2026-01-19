package modes

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aliancn/logcmd/internal/ui/common"
	"github.com/aliancn/logcmd/internal/ui/services/formatter"
	"github.com/aliancn/logcmd/internal/ui/services/highlighter"
)

const maxHighlightBytes = 2 * 1024 * 1024 // 超过该大小的文件不执行语法高亮

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
	returnMode  string
	follow      bool
	returnData  interface{}

	// UI 组件
	viewport viewport.Model

	// 状态
	lines          []string
	yOffset        int
	loaded         bool
	error          error
	active         bool
	useHighlighter bool

	// 布局
	width  int
	height int

	// 样式
	theme  common.Theme
	styles common.Styles

	// 配置
	followInterval time.Duration

	// 服务层
	highlighter *highlighter.ChromaHighlighter
	formatter   *formatter.JSONFormatter
}

// NewLogViewMode 创建日志查看模式
func NewLogViewMode(theme common.Theme, styles common.Styles) *LogViewMode {
	vp := viewport.New(0, 0)

	// 创建highlighter和formatter
	h := highlighter.NewHighlighter()
	h.SetFormat(highlighter.FormatAuto)
	h.SetTheme("monokai")

	f := formatter.NewJSONFormatter(true)

	return &LogViewMode{
		viewport:       vp,
		theme:          theme,
		styles:         styles,
		followInterval: time.Second,
		highlighter:    h,
		formatter:      f,
		useHighlighter: true,
	}
}

// Name 实现 Mode 接口
func (m *LogViewMode) Name() string {
	return "logview"
}

// Activate 实现 Mode 接口
func (m *LogViewMode) Activate() tea.Cmd {
	m.active = true

	// 如果有文件路径，直接返回加载命令（简化逻辑）
	if m.filePath != "" {
		// 确保在激活时重置加载状态
		m.loaded = false
		m.error = nil

		// 如果需要 follow，后续会通过 Update 处理
		if m.follow {
			// 返回批处理命令
			return tea.Batch(m.loadFileCmd(), m.followTickCmd())
		}

		// 直接返回加载命令
		return m.loadFileCmd()
	}

	// 如果没有文件路径，标记为加载完成以避免卡在加载状态
	m.loaded = true
	m.error = fmt.Errorf("未设置日志文件路径")
	return nil
}

// Deactivate 实现 Mode 接口
func (m *LogViewMode) Deactivate() tea.Cmd {
	m.active = false
	// 清理状态（但保留 filePath 以便下次激活）
	m.lines = nil
	m.yOffset = 0
	m.loaded = false
	m.error = nil
	m.follow = false
	m.returnData = nil
	m.useHighlighter = true
	// 注意：不清理 m.filePath，因为可能需要在下次激活时使用
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
		
		if m.loaded {
			m.updateViewportContent()
		}

		return m, nil

	case fileLoadedMsg:
		// 分割内容为行，避免每次渲染都分割
		m.lines = strings.Split(msg.content, "\n")
		m.useHighlighter = msg.useHighlighter
		m.loaded = true
		m.error = nil

		// 初始化偏移量
		if m.follow {
			m.gotoBottom()
		} else if m.lineNum > 0 {
			m.gotoLine(m.lineNum)
		} else {
			m.yOffset = 0
		}
		
		m.updateViewportContent()
		return m, nil

	case fileLoadFailedMsg:
		m.error = msg.err
		m.loaded = true
		return m, nil

	case followTickMsg:
		if m.follow && m.active && m.filePath != "" {
			return m, tea.Batch(m.loadFileCmd(), m.followTickCmd())
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			// 返回原模式
			mode := m.returnMode
			if mode == "" {
				mode = "search"
			}
			return m, func() tea.Msg {
				return SwitchModeMsg{
					ModeName: mode,
					Data:     m.returnData,
				}
			}
		// 滚动控制
		case "j", "down":
			m.scrollDown(1)
		case "k", "up":
			m.scrollUp(1)
		case "pgdown", " ":
			m.scrollDown(m.viewport.Height)
		case "pgup", "b":
			m.scrollUp(m.viewport.Height)
		case "g", "home":
			m.gotoTop()
		case "G", "end":
			m.gotoBottom()
		}
	}

	// 我们自己接管了滚动逻辑，不再调用 m.viewport.Update(msg) 处理 KeyMsg
	// 但仍然需要调用它来处理其他消息（如鼠标事件等，如果有的话）
	// 不过为了避免冲突，这里我们只在非KeyMsg时调用，或者干脆不调用，
	// 因为我们每次 Update 都手动 SetContent 了。
	// 为了保持 resize 等内部逻辑（虽然我们手动处理了 resize），还是保留它，但这里 KeyMsg 已经被拦截。
	// m.viewport, cmd = m.viewport.Update(msg) 
	
	// 由于我们手动设置 Content 为可见区域，viewport 内部的 offset 应该始终为 0
	// 任何时候都不需要 viewport 自己去滚动
	
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
func (m *LogViewMode) SetFile(filePath string, lineNum int, searchQuery string, returnMode string, follow bool, returnData interface{}) {
	m.filePath = filePath
	m.lineNum = lineNum
	m.searchQuery = searchQuery
	if returnMode == "" {
		returnMode = "search"
	}
	m.returnMode = returnMode
	m.follow = follow
	m.returnData = returnData
	m.loaded = false
	m.error = nil
	m.useHighlighter = true
	m.lines = nil
	m.yOffset = 0
}

// loadFileCmd 加载文件内容
func (m *LogViewMode) loadFileCmd() tea.Cmd {
	filePath := m.filePath // 捕获当前文件路径
	return func() tea.Msg {
		// 添加防御性检查
		if filePath == "" {
			return fileLoadFailedMsg{err: fmt.Errorf("文件路径为空")}
		}

		// 检查文件是否存在
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return fileLoadFailedMsg{err: fmt.Errorf("文件不存在或无法访问: %w", err)}
		}

		// 如果文件太大（超过 100MB），提前警告
		if fileInfo.Size() > 100*1024*1024 {
			return fileLoadFailedMsg{err: fmt.Errorf("文件过大（%.1fMB），无法加载", float64(fileInfo.Size())/(1024*1024))}
		}

		// 读取文件
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fileLoadFailedMsg{err: fmt.Errorf("读取文件失败: %w", err)}
		}

		// 安全地转换为字符串
		content := string(data)
		useHighlighter := len(data) <= maxHighlightBytes

		return fileLoadedMsg{content: content, useHighlighter: useHighlighter}
	}
}

func (m *LogViewMode) followTickCmd() tea.Cmd {
	if !m.follow || m.followInterval <= 0 {
		return nil
	}
	interval := m.followInterval
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return followTickMsg{}
	})
}

// updateViewportContent 计算可见区域并渲染
func (m *LogViewMode) updateViewportContent() {
	if len(m.lines) == 0 {
		m.viewport.SetContent("(空文件)")
		return
	}

	// 确保 viewport 高度已设置
	if m.viewport.Height == 0 {
		return
	}

	// 限制 yOffset 范围
	maxOffset := len(m.lines) - m.viewport.Height
	if maxOffset < 0 {
		maxOffset = 0
	}
	
	if m.yOffset > maxOffset {
		m.yOffset = maxOffset
	}
	if m.yOffset < 0 {
		m.yOffset = 0
	}

	// 计算可见行
	end := m.yOffset + m.viewport.Height
	if end > len(m.lines) {
		end = len(m.lines)
	}
	
	visibleLines := m.lines[m.yOffset:end]
	
	// 渲染可见行
	renderedContent := m.renderLines(visibleLines, m.yOffset+1)
	m.viewport.SetContent(renderedContent)
	m.viewport.SetYOffset(0) // 总是重置为0，因为我们手动控制了内容
}

// renderLines 渲染指定的行片段
func (m *LogViewMode) renderLines(lines []string, startLineNum int) string {
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
		lineNum := startLineNum + i
		isMatchLine := lineNum == m.lineNum

		// 行号
		lineNumText := lineNumStyle.Render(fmt.Sprintf("%d ", lineNum))

		// 应用语法高亮
		displayLine := line
		if m.useHighlighter {
			// 注意：Highlighter 可能需要完整行来正确解析上下文，
			// 但 Chroma 的 HighlightLine 通常可以处理单行（尽管某些多行语法可能受影响）。
			// 为了性能，这是必要的妥协。
			displayLine = m.highlighter.HighlightLine(line, lineNum)
		}

		// 高亮搜索关键词（在语法高亮之后）
		if m.searchQuery != "" && strings.Contains(strings.ToLower(line), strings.ToLower(m.searchQuery)) {
			// 简单的关键词高亮（不区分大小写）
			displayLine = highlightKeyword(displayLine, m.searchQuery, highlightStyle)
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

// 滚动辅助函数

func (m *LogViewMode) scrollDown(lines int) {
	if len(m.lines) == 0 {
		return
	}
	m.yOffset += lines
	m.updateViewportContent()
}

func (m *LogViewMode) scrollUp(lines int) {
	m.yOffset -= lines
	m.updateViewportContent()
}

func (m *LogViewMode) gotoTop() {
	m.yOffset = 0
	m.updateViewportContent()
}

func (m *LogViewMode) gotoBottom() {
	if len(m.lines) == 0 {
		return
	}
	m.yOffset = len(m.lines) - m.viewport.Height
	if m.yOffset < 0 {
		m.yOffset = 0
	}
	m.updateViewportContent()
}

// gotoLine 跳转到指定行
func (m *LogViewMode) gotoLine(lineNum int) {
	if lineNum <= 0 {
		return
	}

	if lineNum > len(m.lines) {
		lineNum = len(m.lines)
	}

	// 计算目标行的偏移
	// 让目标行显示在viewport的中间位置
	targetOffset := lineNum - m.viewport.Height/2
	if targetOffset < 0 {
		targetOffset = 0
	}

	m.yOffset = targetOffset
	m.updateViewportContent()
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
	if m.follow {
		status += " · 实时"
	}
	if !m.useHighlighter {
		status += " · 高亮关闭"
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
	modeName := displayModeName(m.returnMode)

	// 根据宽度调整显示的快捷键
	var hints string
	if m.width >= 120 {
		// 宽屏：显示所有快捷键
		hints = fmt.Sprintf("j/k:滚动 · gg/G:首/尾 · PgUp/Dn:翻页 · q/Esc:返回%s", modeName)
	} else if m.width >= 90 {
		// 中等宽度
		hints = fmt.Sprintf("j/k:滚动 · gg/G:首/尾 · q/Esc:返回%s", modeName)
	} else {
		// 窄屏：只显示基础功能
		hints = fmt.Sprintf("↑↓:滚动 · q/Esc:返回%s", modeName)
	}

	if m.follow {
		hints += " · 实时刷新中"
	}

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

	debugStyle := lipgloss.NewStyle().
		Foreground(m.theme.TextMuted)

	// 显示加载状态和调试信息
	view += loadingStyle.Render("⏳ 正在加载日志文件...") + "\n\n"
	view += debugStyle.Render(fmt.Sprintf("文件路径: %s", m.filePath)) + "\n"
	view += debugStyle.Render(fmt.Sprintf("激活状态: %v", m.active)) + "\n"
	view += debugStyle.Render(fmt.Sprintf("加载状态: %v", m.loaded))
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
	content        string
	useHighlighter bool
}

type fileLoadFailedMsg struct {
	err error
}

type followTickMsg struct{}

func displayModeName(mode string) string {
	switch mode {
	case "task":
		return "任务"
	case "project":
		return "项目"
	case "stats":
		return "统计"
	case "command":
		return "命令"
	case "logview":
		return "日志"
	default:
		return "搜索"
	}
}
