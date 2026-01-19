package logviewer

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aliancn/logcmd/internal/model"
	"github.com/aliancn/logcmd/internal/ui/common"
	"github.com/aliancn/logcmd/internal/ui/components/panel"
	"github.com/aliancn/logcmd/internal/ui/services/formatter"
	"github.com/aliancn/logcmd/internal/ui/services/highlighter"
	"github.com/aliancn/logcmd/internal/ui/services/reader"
)

// Model 展示日志内容。
type Model struct {
	viewport    viewport.Model
	panel       *panel.Panel
	history     *model.CommandHistory
	filePath    string // 直接指定的文件路径（优先级高于 history）
	content     string // 小文件的完整内容（向后兼容）
	width       int
	height      int
	theme       common.Theme
	styles      common.Styles
	keys        keyMap
	searchInput textinput.Model
	searching   bool
	lastQuery   string
	
	gotoInput   textinput.Model
	goingToLine bool

	statusMsg   string
	prettyJson  bool // JSON 格式化模式

	// 服务层
	highlighter *highlighter.ChromaHighlighter
	formatter   *formatter.JSONFormatter

	// 虚拟滚动（大文件）
	chunkedReader  *reader.ChunkedReader
	usesChunked    bool // 是否使用分块读取
	cachedLines    []string
	cacheStartLine int // 缓存的起始行号（0-based）
	cacheEndLine   int // 缓存的结束行号（0-based）
	bufferSize     int // 预加载缓冲区大小（行数）
	totalLines     int // 文件总行数
	indexing       bool // 是否正在建立索引
	indexProgress  float64 // 索引进度 0.0-1.0

	// 交互功能
	showLineNumbers bool // 是否显示行号
	softWrap        bool // 是否软换行

	// 搜索增强
	searchMatches []int // 所有匹配行的行号
	currentMatch  int   // 当前匹配项索引
}

// ContentLoadedMsg 表示日志内容已加载。
type ContentLoadedMsg struct {
	HistoryID int
	Content   string
}

// IndexBuiltMsg 表示文件索引已构建完成
type IndexBuiltMsg struct {
	HistoryID int
	Reader    *reader.ChunkedReader
	TotalLines int
}

// LinesLoadedMsg 表示行数据已加载
type LinesLoadedMsg struct {
	Lines      []string
	StartLine  int
	EndLine    int
}

// IndexProgressMsg 表示索引进度更新
type IndexProgressMsg struct {
	Progress float64
}

// BackMsg 表示用户请求返回上一视图。
type BackMsg struct{}

// New 创建日志查看器。
func New(theme common.Theme, styles common.Styles) Model {
	vp := viewport.New(0, 0)
	input := textinput.New()
	input.Prompt = "/ "
	input.Placeholder = "输入搜索关键词"

	gInput := textinput.New()
	gInput.Prompt = ":"
	gInput.Placeholder = "行号"
	gInput.CharLimit = 10
	gInput.Validate = func(s string) error {
		if s == "" {
			return nil
		}
		for _, c := range s {
			if c < '0' || c > '9' {
				return fmt.Errorf("number required")
			}
		}
		return nil
	}

	// 创建Panel布局容器
	p := panel.NewDefault("", theme, styles)

	// 创建highlighter和formatter
	h := highlighter.NewHighlighter()
	h.SetFormat(highlighter.FormatAuto)
	h.SetTheme("monokai")

	f := formatter.NewJSONFormatter(true) // 启用彩色输出

	return Model{
		viewport:        vp,
		panel:           p,
		theme:           theme,
		styles:          styles,
		keys:            newKeyMap(),
		searchInput:     input,
		gotoInput:       gInput,
		highlighter:     h,
		formatter:       f,
		bufferSize:      100, // 默认预加载100行
		showLineNumbers: true, // 默认显示行号
		softWrap:        false, // 默认不换行
	}
}

// Init 实现 tea.Model。
func (m Model) Init() tea.Cmd {
	return nil
}

// SetFile 指定要查看的日志文件路径（不依赖历史记录）。
func (m *Model) SetFile(path string) {
	// 清理旧的chunkedReader
	if m.chunkedReader != nil {
		m.chunkedReader.Close()
		m.chunkedReader = nil
	}

	m.history = nil
	m.filePath = path
	m.content = ""
	m.viewport.SetContent("")
	m.viewport.GotoTop()
	m.statusMsg = ""
	m.usesChunked = false
	m.cachedLines = nil
	m.cacheStartLine = 0
	m.cacheEndLine = 0
	m.totalLines = 0
	m.indexing = false
}

// SetHistory 指定要查看的历史记录。
func (m *Model) SetHistory(history *model.CommandHistory) {
	// 清理旧的chunkedReader
	if m.chunkedReader != nil {
		m.chunkedReader.Close()
		m.chunkedReader = nil
	}

	m.history = history
	m.content = ""
	m.viewport.SetContent("")
	m.viewport.GotoTop()
	m.statusMsg = ""
	m.usesChunked = false
	m.cachedLines = nil
	m.cacheStartLine = 0
	m.cacheEndLine = 0
	m.totalLines = 0
	m.indexing = false
}

// Reset 清理状态。
func (m *Model) Reset() {
	// 清理chunkedReader
	if m.chunkedReader != nil {
		m.chunkedReader.Close()
		m.chunkedReader = nil
	}

	m.history = nil
	m.filePath = ""
	m.content = ""
	m.viewport.SetContent("")
	m.statusMsg = ""
	m.usesChunked = false
	m.cachedLines = nil
	m.cacheStartLine = 0
	m.cacheEndLine = 0
	m.totalLines = 0
	m.indexing = false
}

// SetSize 调整 viewport 大小。
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height

	// 构建header
	var header string
	if m.history != nil {
		headerStyle := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true)
		mutedStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)

		var headerBuilder strings.Builder
		headerBuilder.WriteString(headerStyle.Render(fmt.Sprintf("#%d %s", m.history.ID, m.history.Command)))
		headerBuilder.WriteString("\n")
		headerBuilder.WriteString(mutedStyle.Render(fmt.Sprintf("日志文件: %s", m.history.LogFilePath)))
		headerBuilder.WriteString("\n")
		headerBuilder.WriteString(mutedStyle.Render(fmt.Sprintf("开始时间: %s", m.history.StartTime.Format(time.RFC3339))))
		header = headerBuilder.String()
	} else if m.filePath != "" {
		headerStyle := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true)
		var headerBuilder strings.Builder
		headerBuilder.WriteString(headerStyle.Render(fmt.Sprintf("日志文件: %s", m.filePath)))
		header = headerBuilder.String()
	}

	// 构建footer
	var footer string
	if m.searching {
		footer = m.searchInput.View()
	} else if m.goingToLine {
		footer = m.gotoInput.View()
	} else {
		// 构建完整的快捷键提示
		hints := m.buildKeyHints()
		if m.statusMsg != "" {
			footer = common.JoinKeyHelps(m.statusMsg, hints)
		} else {
			footer = hints
		}
	}

	// 设置Panel的header和footer
	m.panel.SetHeader(header)
	m.panel.SetFooter(footer)

	// 设置Panel尺寸，Panel会自动计算内容区域（已扣除header和footer）
	m.panel.SetSize(width, height)

	// 获取Panel计算后的精确内容尺寸
	contentW, contentH := m.panel.GetContentSize()

	// 确保最小尺寸
	if contentW < 30 {
		contentW = 30
	}
	if contentH < 5 {
		contentH = 5
	}

	// 设置viewport使用精确的内容尺寸
	m.viewport.Width = contentW
	m.viewport.Height = contentH
}

// LoadContentCmd 读取日志文件。
func (m Model) LoadContentCmd() tea.Cmd {
	var path string
	var historyID int

	if m.filePath != "" {
		path = m.filePath
		historyID = 0 // 0 表示非历史记录关联
	} else if m.history != nil && m.history.LogFilePath != "" {
		path = m.history.LogFilePath
		historyID = m.history.ID
	} else {
		return nil
	}

	return func() tea.Msg {
		// 先获取文件信息
		fileInfo, err := os.Stat(path)
		if err != nil {
			return common.ErrorMsg{Err: fmt.Errorf("获取文件信息失败: %w", err)}
		}

		// 判断是否需要使用分块读取（大于10MB）
		const chunkThreshold = 10 * 1024 * 1024 // 10MB

		if fileInfo.Size() > chunkThreshold {
			// 大文件：使用分块读取
			chunkedReader, err := reader.NewChunkedReader(path, reader.DefaultConfig())
			if err != nil {
				// 分块读取失败，降级到全量读取
				data, readErr := os.ReadFile(path)
				if readErr != nil {
					return common.ErrorMsg{Err: fmt.Errorf("读取日志失败: %w", readErr)}
				}
				return ContentLoadedMsg{
					HistoryID: historyID,
					Content:   string(data),
				}
			}

			// 异步构建索引
			err = chunkedReader.BuildIndex()
			if err != nil {
				chunkedReader.Close()
				// 索引构建失败，降级到全量读取
				data, readErr := os.ReadFile(path)
				if readErr != nil {
					return common.ErrorMsg{Err: fmt.Errorf("读取日志失败: %w", readErr)}
				}
				return ContentLoadedMsg{
					HistoryID: historyID,
					Content:   string(data),
				}
			}

			return IndexBuiltMsg{
				HistoryID:  historyID,
				Reader:     chunkedReader,
				TotalLines: chunkedReader.TotalLines(),
			}
		}

		// 小文件：直接全量读取
		data, err := os.ReadFile(path)
		if err != nil {
			return common.ErrorMsg{Err: fmt.Errorf("读取日志失败: %w", err)}
		}
		return ContentLoadedMsg{
			HistoryID: historyID,
			Content:   string(data),
		}
	}
}

// loadVisibleLinesCmd 加载可见区域的行
func (m Model) loadVisibleLinesCmd() tea.Cmd {
	if !m.usesChunked || m.chunkedReader == nil {
		return nil
	}

	return func() tea.Msg {
		// 计算需要加载的行范围
		visibleStart := m.viewport.YOffset
		visibleEnd := visibleStart + m.viewport.Height

		// 添加缓冲区
		start := visibleStart - m.bufferSize
		if start < 0 {
			start = 0
		}

		end := visibleEnd + m.bufferSize
		if end >= m.totalLines {
			end = m.totalLines - 1
		}

		// 读取行
		lines, err := m.chunkedReader.ReadLines(start, end)
		if err != nil {
			return common.ErrorMsg{Err: fmt.Errorf("加载行失败: %w", err)}
		}

		return LinesLoadedMsg{
			Lines:     lines,
			StartLine: start,
			EndLine:   end,
		}
	}
}

// needsReload 检查是否需要重新加载行
func (m Model) needsReload() bool {
	if !m.usesChunked {
		return false
	}

	offset := m.viewport.YOffset
	threshold := m.bufferSize / 2

	// 如果接近缓存边界，需要重新加载
	return offset < m.cacheStartLine+threshold ||
		offset+m.viewport.Height > m.cacheEndLine-threshold
}

// Update 处理消息。
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
	case tea.KeyMsg:
		if m.searching {
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			cmds = append(cmds, cmd)
			if msg.Type == tea.KeyEnter {
				m.searching = false
				m.lastQuery = strings.TrimSpace(m.searchInput.Value())
				m.searchInput.SetValue("")
				m.searchInput.Blur()
				m.jumpToQuery(m.lastQuery)
				m.updateFooter() // 更新footer显示n/N导航提示
				// 如果是大文件模式，需要加载匹配行
				if m.usesChunked && len(m.searchMatches) > 0 {
					cmds = append(cmds, m.loadVisibleLinesCmd())
				}
			} else if msg.Type == tea.KeyEsc {
				m.searching = false
				m.searchInput.Blur()
				m.updateFooter()
			}
			return m, tea.Batch(cmds...)
		}
		if m.goingToLine {
			var cmd tea.Cmd
			m.gotoInput, cmd = m.gotoInput.Update(msg)
			cmds = append(cmds, cmd)
			if msg.Type == tea.KeyEnter {
				m.goingToLine = false
				val := strings.TrimSpace(m.gotoInput.Value())
				m.gotoInput.SetValue("")
				m.gotoInput.Blur()
				
				if val != "" {
					var lineNum int
					_, err := fmt.Sscanf(val, "%d", &lineNum)
					if err == nil {
						m.JumpToLine(lineNum)
						if m.usesChunked {
							cmds = append(cmds, m.loadVisibleLinesCmd())
						}
					}
				}
				m.updateFooter()
			} else if msg.Type == tea.KeyEsc {
				m.goingToLine = false
				m.gotoInput.Blur()
				m.updateFooter()
			}
			return m, tea.Batch(cmds...)
		}
		switch {
		case key.Matches(msg, m.keys.Back):
			return m, func() tea.Msg { return BackMsg{} }
		case key.Matches(msg, m.keys.Search):
			m.searching = true
			m.searchInput.SetValue("")
			m.searchInput.Focus()
			m.updateFooter() // 显示搜索输入框
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keys.GotoLine):
			m.goingToLine = true
			m.gotoInput.SetValue("")
			m.gotoInput.Focus()
			m.updateFooter()
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keys.ToggleJSON):
			m.prettyJson = !m.prettyJson
			// Re-render content
			if m.usesChunked {
				highlighted := m.highlightCachedLines()
				m.viewport.SetContent(highlighted)
			} else {
				highlighted := m.highlightContent(m.content)
				m.viewport.SetContent(highlighted)
			}
			status := "JSON格式化: 关闭"
			if m.prettyJson {
				status = "JSON格式化: 开启"
			}
			m.statusMsg = status
			m.updateFooter()
		case key.Matches(msg, m.keys.ToggleLineNumbers):
			m.showLineNumbers = !m.showLineNumbers
			// Re-render content
			if m.usesChunked {
				highlighted := m.highlightCachedLines()
				m.viewport.SetContent(highlighted)
			} else {
				highlighted := m.highlightContent(m.content)
				m.viewport.SetContent(highlighted)
			}
			status := "行号显示: 关闭"
			if m.showLineNumbers {
				status = "行号显示: 开启"
			}
			m.statusMsg = status
			m.updateFooter()
		case key.Matches(msg, m.keys.ToggleWrap):
			m.softWrap = !m.softWrap
			status := "软换行: 关闭"
			if m.softWrap {
				status = "软换行: 开启"
			}
			m.statusMsg = status
			m.updateFooter()
			// TODO: 实现软换行逻辑
		case key.Matches(msg, m.keys.NextMatch):
			if len(m.searchMatches) > 0 {
				m.currentMatch = (m.currentMatch + 1) % len(m.searchMatches)
				m.viewport.SetYOffset(m.searchMatches[m.currentMatch])
				m.statusMsg = fmt.Sprintf("匹配 %d/%d", m.currentMatch+1, len(m.searchMatches))
				m.updateFooter()
				// 如果是大文件模式，需要加载匹配行
				if m.usesChunked && m.needsReload() {
					cmds = append(cmds, m.loadVisibleLinesCmd())
				}
			} else {
				m.statusMsg = "无匹配项"
				m.updateFooter()
			}
		case key.Matches(msg, m.keys.PrevMatch):
			if len(m.searchMatches) > 0 {
				m.currentMatch = (m.currentMatch - 1 + len(m.searchMatches)) % len(m.searchMatches)
				m.viewport.SetYOffset(m.searchMatches[m.currentMatch])
				m.statusMsg = fmt.Sprintf("匹配 %d/%d", m.currentMatch+1, len(m.searchMatches))
				m.updateFooter()
				// 如果是大文件模式，需要加载匹配行
				if m.usesChunked && m.needsReload() {
					cmds = append(cmds, m.loadVisibleLinesCmd())
				}
			} else {
				m.statusMsg = "无匹配项"
				m.updateFooter()
			}
		case key.Matches(msg, m.keys.Down):
			prevOffset := m.viewport.YOffset
			m.viewport.LineDown(1)
			if m.viewport.YOffset != prevOffset && m.needsReload() {
				cmds = append(cmds, m.loadVisibleLinesCmd())
			}
		case key.Matches(msg, m.keys.Up):
			prevOffset := m.viewport.YOffset
			m.viewport.LineUp(1)
			if m.viewport.YOffset != prevOffset && m.needsReload() {
				cmds = append(cmds, m.loadVisibleLinesCmd())
			}
		case key.Matches(msg, m.keys.PageDown):
			prevOffset := m.viewport.YOffset
			m.viewport.ViewDown()
			if m.viewport.YOffset != prevOffset && m.needsReload() {
				cmds = append(cmds, m.loadVisibleLinesCmd())
			}
		case key.Matches(msg, m.keys.PageUp):
			prevOffset := m.viewport.YOffset
			m.viewport.ViewUp()
			if m.viewport.YOffset != prevOffset && m.needsReload() {
				cmds = append(cmds, m.loadVisibleLinesCmd())
			}
		case key.Matches(msg, m.keys.Top):
			m.viewport.GotoTop()
			if m.needsReload() {
				cmds = append(cmds, m.loadVisibleLinesCmd())
			}
		case key.Matches(msg, m.keys.Bottom):
			m.viewport.GotoBottom()
			if m.needsReload() {
				cmds = append(cmds, m.loadVisibleLinesCmd())
			}
		}
	case ContentLoadedMsg:
		if (m.history != nil && msg.HistoryID != m.history.ID) && (m.filePath != "" && msg.HistoryID != 0) {
			break
		}
		// 小文件模式
		m.usesChunked = false
		m.content = msg.Content // Keep raw content for search/logic

		// Apply Highlighting
		highlighted := m.highlightContent(msg.Content)

		m.viewport.SetContent(highlighted)
		m.viewport.GotoTop()
		m.statusMsg = fmt.Sprintf("日志加载完成（%d 字节）", len(msg.Content))
		m.updateFooter()

	case IndexBuiltMsg:
		if (m.history != nil && msg.HistoryID != m.history.ID) && (m.filePath != "" && msg.HistoryID != 0) {
			break
		}
		// 大文件模式
		m.usesChunked = true
		m.chunkedReader = msg.Reader
		m.totalLines = msg.TotalLines
		m.indexing = false
		m.statusMsg = fmt.Sprintf("索引构建完成（%d 行）", m.totalLines)
		m.updateFooter()

		// 加载初始可见行
		cmds = append(cmds, m.loadVisibleLinesCmd())

	case LinesLoadedMsg:
		if !m.usesChunked {
			break
		}
		// 更新缓存
		m.cachedLines = msg.Lines
		m.cacheStartLine = msg.StartLine
		m.cacheEndLine = msg.EndLine

		// 高亮并设置内容
		highlighted := m.highlightCachedLines()
		m.viewport.SetContent(highlighted)

	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// highlightContent applies syntax highlighting to log content
func (m Model) highlightContent(content string) string {
	lines := strings.Split(content, "\n")
	var sb strings.Builder

	// 行号样式
	lineNumStyle := lipgloss.NewStyle().
		Foreground(m.theme.TextMuted).
		Width(6).
		Align(lipgloss.Right)

	for i, line := range lines {
		if i > 0 {
			sb.WriteString("\n")
		}

		// 添加行号（如果启用）
		if m.showLineNumbers {
			lineNumText := lineNumStyle.Render(fmt.Sprintf("%d ", i+1))
			sb.WriteString(lineNumText)
			sb.WriteString("  ")
		}

		// JSON Formatting
		if m.prettyJson {
			// 尝试从行中提取JSON
			prefix, jsonPart, suffix, found := formatter.ExtractJSON(line)
			if found {
				// 格式化JSON部分
				formattedJSON, err := m.formatter.Format(jsonPart)
				if err == nil {
					// 成功格式化JSON
					if len(prefix) > 0 {
						// 对前缀应用高亮
						sb.WriteString(m.highlighter.HighlightLine(prefix, i+1))
					}
					sb.WriteString(formattedJSON)
					if len(suffix) > 0 {
						sb.WriteString(suffix)
					}
					continue
				}
			}
		}

		// 使用Chroma进行语法高亮
		highlighted := m.highlighter.HighlightLine(line, i+1)
		sb.WriteString(highlighted)
	}
	return sb.String()
}

// highlightCachedLines 对缓存的行进行高亮
func (m Model) highlightCachedLines() string {
	var sb strings.Builder

	// 行号样式
	lineNumStyle := lipgloss.NewStyle().
		Foreground(m.theme.TextMuted).
		Width(6).
		Align(lipgloss.Right)

	for i, line := range m.cachedLines {
		if i > 0 {
			sb.WriteString("\n")
		}

		lineNum := m.cacheStartLine + i + 1

		// 添加行号（如果启用）
		if m.showLineNumbers {
			lineNumText := lineNumStyle.Render(fmt.Sprintf("%d ", lineNum))
			sb.WriteString(lineNumText)
			sb.WriteString("  ")
		}

		// JSON Formatting
		if m.prettyJson {
			// 尝试从行中提取JSON
			prefix, jsonPart, suffix, found := formatter.ExtractJSON(line)
			if found {
				// 格式化JSON部分
				formattedJSON, err := m.formatter.Format(jsonPart)
				if err == nil {
					// 成功格式化JSON
					if len(prefix) > 0 {
						// 对前缀应用高亮
						sb.WriteString(m.highlighter.HighlightLine(prefix, lineNum))
					}
					sb.WriteString(formattedJSON)
					if len(suffix) > 0 {
						sb.WriteString(suffix)
					}
					continue
				}
			}
		}

		// 使用Chroma进行语法高亮
		highlighted := m.highlighter.HighlightLine(line, lineNum)
		sb.WriteString(highlighted)
	}
	return sb.String()
}

// View 渲染日志。
func (m Model) View() string {
	if m.history == nil {
		return m.panel.RenderEmpty("选择一条历史记录查看日志")
	}

	// 使用Panel渲染viewport内容
	// header和footer已经在SetSize中设置好了
	return m.panel.Render(m.viewport.View())
}

// buildKeyHints 构建快捷键提示
func (m Model) buildKeyHints() string {
	// 根据窗口宽度调整显示的快捷键
	hints := []string{}

	// 基础导航（始终显示）
	hints = append(hints, "j/k:滚动")
	hints = append(hints, "gg/G:首/尾")

	// 搜索功能
	if len(m.searchMatches) > 0 {
		hints = append(hints, "n/N:下/上个匹配")
	} else {
		hints = append(hints, "/:搜索")
	}

	// 根据窗口宽度决定显示更多快捷键
	if m.width >= 120 {
		// 宽屏：显示所有快捷键
		hints = append(hints, "Ctrl+L:行号")
		hints = append(hints, "Ctrl+J:JSON")
		hints = append(hints, "Ctrl+W:换行")
		hints = append(hints, "Esc:返回")
	} else if m.width >= 90 {
		// 中等宽度：显示主要功能
		hints = append(hints, "Ctrl+L:行号")
		hints = append(hints, "Ctrl+J:JSON")
		hints = append(hints, "Esc:返回")
	} else {
		// 窄屏：只显示基础功能
		hints = append(hints, "Esc:返回")
	}

	return strings.Join(hints, " · ")
}

// updateFooter 更新footer显示
func (m *Model) updateFooter() {
	var footer string
	if m.searching {
		footer = m.searchInput.View()
	} else if m.goingToLine {
		footer = m.gotoInput.View()
	} else {
		hints := m.buildKeyHints()
		if m.statusMsg != "" {
			footer = common.JoinKeyHelps(m.statusMsg, hints)
		} else {
			footer = hints
		}
	}
	m.panel.SetFooter(footer)
}

// jumpToQuery 在日志中查找文本并存储所有匹配项。
func (m *Model) jumpToQuery(query string) {
	if query == "" {
		m.statusMsg = "未输入搜索关键词"
		m.searchMatches = nil
		m.currentMatch = 0
		return
	}

	lowerQuery := strings.ToLower(query)
	m.searchMatches = nil
	m.currentMatch = 0

	// 小文件模式：在完整内容中搜索
	if !m.usesChunked && m.content != "" {
		lines := strings.Split(m.content, "\n")
		for idx, line := range lines {
			if strings.Contains(strings.ToLower(line), lowerQuery) {
				m.searchMatches = append(m.searchMatches, idx)
			}
		}

		if len(m.searchMatches) > 0 {
			// 找到从当前位置开始的第一个匹配
			currentOffset := m.viewport.YOffset
			for i, lineIdx := range m.searchMatches {
				if lineIdx >= currentOffset {
					m.currentMatch = i
					m.viewport.SetYOffset(lineIdx)
					m.statusMsg = fmt.Sprintf("找到 %d 个匹配，当前 %d/%d", len(m.searchMatches), i+1, len(m.searchMatches))
					return
				}
			}
			// 如果所有匹配都在当前位置之前，跳转到第一个匹配
			m.currentMatch = 0
			m.viewport.SetYOffset(m.searchMatches[0])
			m.statusMsg = fmt.Sprintf("找到 %d 个匹配，当前 1/%d", len(m.searchMatches), len(m.searchMatches))
		} else {
			m.statusMsg = "未找到匹配"
		}
		return
	}

	// 大文件模式：在整个文件中搜索（需要读取所有行）
	if m.usesChunked && m.chunkedReader != nil {
		// 为了搜索功能，我们需要扫描整个文件
		// 这可能比较耗时，但对于搜索功能是必要的
		totalLines := m.totalLines
		batchSize := 1000 // 每次读取1000行

		for start := 0; start < totalLines; start += batchSize {
			end := start + batchSize - 1
			if end >= totalLines {
				end = totalLines - 1
			}

			lines, err := m.chunkedReader.ReadLines(start, end)
			if err != nil {
				m.statusMsg = fmt.Sprintf("搜索出错: %v", err)
				return
			}

			for i, line := range lines {
				if strings.Contains(strings.ToLower(line), lowerQuery) {
					m.searchMatches = append(m.searchMatches, start+i)
				}
			}
		}

		if len(m.searchMatches) > 0 {
			// 找到从当前位置开始的第一个匹配
			currentOffset := m.viewport.YOffset
			for i, lineIdx := range m.searchMatches {
				if lineIdx >= currentOffset {
					m.currentMatch = i
					m.viewport.SetYOffset(lineIdx)
					// 需要加载包含该匹配行的内容
					cmd := m.loadVisibleLinesCmd()
					if cmd != nil {
						// 这里无法直接执行cmd，但会在下一次Update中触发加载
					}
					m.statusMsg = fmt.Sprintf("找到 %d 个匹配，当前 %d/%d", len(m.searchMatches), i+1, len(m.searchMatches))
					return
				}
			}
			// 如果所有匹配都在当前位置之前，跳转到第一个匹配
			m.currentMatch = 0
			m.viewport.SetYOffset(m.searchMatches[0])
			m.statusMsg = fmt.Sprintf("找到 %d 个匹配，当前 1/%d", len(m.searchMatches), len(m.searchMatches))
		} else {
			m.statusMsg = "未找到匹配"
		}
	}
}

// JumpToLine jumps to a specific line number (1-based)
func (m *Model) JumpToLine(line int) {
	if line < 1 {
		line = 1
	}
	// Adjust for 0-based index
	idx := line - 1

	// Ensure we don't go out of bounds (rough check, viewport handles it well usually)
	// But getting total lines requires splitting content which might be heavy?
	// Viewport.SetYOffset handles validation mostly.
	m.viewport.SetYOffset(idx)
	m.statusMsg = fmt.Sprintf("跳转到行: %d", line)
}

// PerformSearch performs a search with the given query
func (m *Model) PerformSearch(query string) tea.Cmd {
	m.lastQuery = query
	m.jumpToQuery(query)
	
	// 如果是大文件模式且有匹配，可能需要加载行
	if m.usesChunked && len(m.searchMatches) > 0 {
		return m.loadVisibleLinesCmd()
	}
	return nil
}

type keyMap struct {
	Back              key.Binding
	Search            key.Binding
	GotoLine          key.Binding
	Down              key.Binding
	Up                key.Binding
	PageDown          key.Binding
	PageUp            key.Binding
	Top               key.Binding
	Bottom            key.Binding
	ToggleJSON        key.Binding
	ToggleLineNumbers key.Binding
	ToggleWrap        key.Binding
	NextMatch         key.Binding
	PrevMatch         key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "返回历史"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "搜索"),
		),
		GotoLine: key.NewBinding(
			key.WithKeys(":"),
			key.WithHelp(":", "跳转行"),
		),
		ToggleJSON: key.NewBinding(
			key.WithKeys("ctrl+j"),
			key.WithHelp("ctrl+j", "JSON格式化"),
		),
		ToggleLineNumbers: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("ctrl+l", "切换行号"),
		),
		ToggleWrap: key.NewBinding(
			key.WithKeys("ctrl+w"),
			key.WithHelp("ctrl+w", "软换行"),
		),
		NextMatch: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "下一个匹配"),
		),
		PrevMatch: key.NewBinding(
			key.WithKeys("N"),
			key.WithHelp("N", "上一个匹配"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "下一行"),
		),
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "上一行"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("ctrl+f", "pgdown"),
			key.WithHelp("Ctrl+F/PgDn", "下一页"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("ctrl+b", "pgup"),
			key.WithHelp("Ctrl+B/PgUp", "上一页"),
		),
		Top: key.NewBinding(
			key.WithKeys("g", "home"),
			key.WithHelp("g/home", "顶端"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G", "end"),
			key.WithHelp("G/end", "底部"),
		),
	}
}
