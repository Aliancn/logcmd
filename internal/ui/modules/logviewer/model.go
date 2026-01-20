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
	"github.com/cespare/xxhash/v2"
	lru "github.com/hashicorp/golang-lru/v2"

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
	err         error
	prettyJson  bool // JSON 格式化模式
	loading     bool // 是否正在加载

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
	searchMatches    []int  // 所有匹配行的行号
	currentMatch     int    // 当前匹配项索引
	searchQuery      string // 当前搜索关键词
	highlightMatches bool   // 是否高亮匹配（默认true）

	// 性能优化：高亮缓存
	highlightCache *lru.Cache[int, string] // 行号 -> 高亮后的内容（LRU限制大小）

	// 渲染优化
	lastRenderedHash uint64 // 上次渲染内容的hash
	needsFullRender  bool   // 是否需要完整重渲染

	// 惰性高亮优化
	visibleStart  int  // viewport可见起始行
	visibleEnd    int  // viewport可见结束行
	highlightLazy bool // 是否启用惰性高亮（默认true）
}

// ContentLoadedMsg 表示日志内容已加载。
type ContentLoadedMsg struct {
	HistoryID int
	Content   string
}

// PartialContentLoadedMsg 表示部分日志内容已快速加载（用于首屏显示）
type PartialContentLoadedMsg struct {
	HistoryID      int
	Lines          []string
	IsFullFile     bool // 是否为完整文件（小文件）
	Reader         *reader.ChunkedReader // 大文件时传递reader
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

	// 初始化LRU缓存（限制2000行）
	highlightCache, _ := lru.New[int, string](2000)

	return Model{
		viewport:         vp,
		panel:            p,
		theme:            theme,
		styles:           styles,
		keys:             newKeyMap(),
		searchInput:      input,
		gotoInput:        gInput,
		highlighter:      h,
		formatter:        f,
		bufferSize:       100,    // 默认预加载100行
		showLineNumbers:  true,   // 默认显示行号
		softWrap:         false,  // 默认不换行
		highlightCache:   highlightCache,
		highlightLazy:    true,   // 默认启用惰性高亮
		highlightMatches: true,   // 默认启用搜索高亮
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
	m.filePath = path // 设置文件路径
	m.viewport.GotoTop()
	m.statusMsg = "加载中..."
	m.loading = true
	m.usesChunked = false
	m.cachedLines = nil
	m.cacheStartLine = 0
	m.cacheEndLine = 0
	m.totalLines = 0
	m.indexing = false
	m.highlightCache.Purge() // 清除高亮缓存

	// 重置渲染状态，确保新内容一定会被渲染
	m.lastRenderedHash = 0
	m.needsFullRender = true
}

// SetHistory 指定要查看的历史记录。
func (m *Model) SetHistory(history *model.CommandHistory) {
	// 清理旧的chunkedReader
	if m.chunkedReader != nil {
		m.chunkedReader.Close()
		m.chunkedReader = nil
	}

	m.history = history
	m.filePath = "" // 清除文件路径
	m.err = nil
	m.content = ""
	m.viewport.SetContent("")
	m.viewport.GotoTop()
	m.statusMsg = "加载中..."
	m.loading = true
	m.usesChunked = false
	m.cachedLines = nil
	m.cacheStartLine = 0
	m.cacheEndLine = 0
	m.totalLines = 0
	m.indexing = false
	m.highlightCache.Purge() // 清除高亮缓存

	// 重置渲染状态，确保新内容一定会被渲染
	m.lastRenderedHash = 0
	m.needsFullRender = true
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
	m.err = nil
	m.content = ""
	m.viewport.SetContent("")
	m.statusMsg = ""
	m.loading = false
	m.usesChunked = false
	m.cachedLines = nil
	m.cacheStartLine = 0
	m.cacheEndLine = 0
	m.totalLines = 0
	m.indexing = false
	m.highlightCache.Purge() // 清除高亮缓存

	// 重置渲染状态
	m.lastRenderedHash = 0
	m.needsFullRender = true
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

		// 判断是否需要使用分块读取
		// 注意：这里使用较小的阈值（2MB）是因为：
		// 1. 避免对大行数文件（如2万行日志）进行全量语法高亮导致卡顿
		// 2. 分块模式有缓存优化，性能更好
		// 3. 渐进式加载提供更好的用户体验
		const chunkThreshold = 2 * 1024 * 1024 // 2MB

		if fileInfo.Size() > chunkThreshold {
			// 大文件：使用渐进式加载策略
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

			// 快速读取文件头部（1000行）用于首屏显示
			const quickLoadLines = 1000
			lines, err := chunkedReader.QuickReadLines(quickLoadLines)
			if err != nil {
				chunkedReader.Close()
				// 快速读取失败，降级到全量读取
				data, readErr := os.ReadFile(path)
				if readErr != nil {
					return common.ErrorMsg{Err: fmt.Errorf("读取日志失败: %w", readErr)}
				}
				return ContentLoadedMsg{
					HistoryID: historyID,
					Content:   string(data),
				}
			}

			// 返回部分内容，稍后会在后台构建索引
			return PartialContentLoadedMsg{
				HistoryID:  historyID,
				Lines:      lines,
				IsFullFile: false,
				Reader:     chunkedReader,
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

// buildIndexCmd 后台构建索引
func (m Model) buildIndexCmd() tea.Cmd {
	if m.chunkedReader == nil {
		return nil
	}

	var historyID int
	if m.history != nil {
		historyID = m.history.ID
	}

	reader := m.chunkedReader

	return func() tea.Msg {
		// 在后台构建索引
		err := reader.BuildIndex()
		if err != nil {
			return common.ErrorMsg{Err: fmt.Errorf("索引构建失败: %w", err)}
		}

		return IndexBuiltMsg{
			HistoryID:  historyID,
			Reader:     reader,
			TotalLines: reader.TotalLines(),
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
	case common.ErrorMsg:
		m.loading = false
		m.err = msg.Err
		m.statusMsg = fmt.Sprintf("错误: %v", msg.Err)
		m.updateFooter()
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
			// 清除高亮缓存，因为格式化选项改变了
			m.highlightCache.Purge()
			m.needsFullRender = true // 强制重新渲染
			// Re-render content (统一使用惰性高亮路径)
			if len(m.cachedLines) > 0 {
				highlighted := m.highlightCachedLines()
				hash := xxhash.Sum64String(highlighted)
				if m.shouldRender(hash) {
					m.viewport.SetContent(highlighted)
				}
			}
			status := "JSON格式化: 关闭"
			if m.prettyJson {
				status = "JSON格式化: 开启"
			}
			m.statusMsg = status
			m.updateFooter()
		case key.Matches(msg, m.keys.ToggleLineNumbers):
			m.showLineNumbers = !m.showLineNumbers
			// 清除高亮缓存，因为行号显示选项改变了
			m.highlightCache.Purge()
			m.needsFullRender = true // 强制重新渲染
			// Re-render content (统一使用惰性高亮路径)
			if len(m.cachedLines) > 0 {
				highlighted := m.highlightCachedLines()
				hash := xxhash.Sum64String(highlighted)
				if m.shouldRender(hash) {
					m.viewport.SetContent(highlighted)
				}
			}
			status := "行号显示: 关闭"
			if m.showLineNumbers {
				status = "行号显示: 开启"
			}
			m.statusMsg = status
			m.updateFooter()
		case key.Matches(msg, m.keys.ToggleWrap):
			m.softWrap = !m.softWrap
			// 清除高亮缓存，因为换行会改变显示
			m.highlightCache.Purge()
			m.needsFullRender = true // 强制重新渲染
			// Re-render content (统一使用惰性高亮路径)
			if len(m.cachedLines) > 0 {
				highlighted := m.highlightCachedLines()
				hash := xxhash.Sum64String(highlighted)
				if m.shouldRender(hash) {
					m.viewport.SetContent(highlighted)
				}
			}
			status := "软换行: 关闭"
			if m.softWrap {
				status = "软换行: 开启"
			}
			m.statusMsg = status
			m.updateFooter()
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
		m.loading = false
		// 小文件也使用惰性高亮路径（性能优化）
		m.usesChunked = false
		m.content = msg.Content // Keep raw content for search/logic

		// 将内容分割为行，使用惰性高亮机制
		lines := strings.Split(msg.Content, "\n")
		m.cachedLines = lines
		m.cacheStartLine = 0
		m.cacheEndLine = len(lines) - 1
		m.totalLines = len(lines)

		// 使用惰性高亮（只高亮可见区域 + 享受LRU缓存）
		highlighted := m.highlightCachedLines()
		hash := xxhash.Sum64String(highlighted)
		if m.shouldRender(hash) {
			m.viewport.SetContent(highlighted)
		}
		m.viewport.GotoTop()
		m.statusMsg = fmt.Sprintf("日志加载完成（%d 行）", len(lines))
		m.updateFooter()

	case PartialContentLoadedMsg:
		if (m.history != nil && msg.HistoryID != m.history.ID) && (m.filePath != "" && msg.HistoryID != 0) {
			break
		}
		m.loading = false

		if msg.IsFullFile {
			// 小文件，也使用惰性高亮路径（性能优化）
			m.usesChunked = false
			m.content = strings.Join(msg.Lines, "\n")

			// 使用惰性高亮机制
			m.cachedLines = msg.Lines
			m.cacheStartLine = 0
			m.cacheEndLine = len(msg.Lines) - 1
			m.totalLines = len(msg.Lines)

			highlighted := m.highlightCachedLines()
			hash := xxhash.Sum64String(highlighted)
			if m.shouldRender(hash) {
				m.viewport.SetContent(highlighted)
			}
			m.viewport.GotoTop()
			m.statusMsg = fmt.Sprintf("日志加载完成（%d 行）", len(msg.Lines))
		} else {
			// 大文件，显示部分内容，准备后台索引
			m.usesChunked = true
			m.chunkedReader = msg.Reader
			m.cachedLines = msg.Lines
			m.cacheStartLine = 0
			m.cacheEndLine = len(msg.Lines) - 1
			m.totalLines = len(msg.Lines) // 临时值，索引完成后会更新
			m.indexing = true

			// 立即显示已加载的内容
			highlighted := m.highlightCachedLines()
			hash := xxhash.Sum64String(highlighted)
			if m.shouldRender(hash) {
				m.viewport.SetContent(highlighted)
			}
			m.viewport.GotoTop()
			m.statusMsg = fmt.Sprintf("快速加载完成（前 %d 行），正在建立索引...", len(msg.Lines))

			// 启动后台索引构建
			cmds = append(cmds, m.buildIndexCmd())
		}
		m.updateFooter()

	case IndexBuiltMsg:
		if (m.history != nil && msg.HistoryID != m.history.ID) && (m.filePath != "" && msg.HistoryID != 0) {
			break
		}
		m.loading = false
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
		hash := xxhash.Sum64String(highlighted)
		if m.shouldRender(hash) {
			m.viewport.SetContent(highlighted)
		}

	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// highlightCachedLines 对缓存的行进行高亮（带缓存优化和惰性高亮）
// 注：原 highlightContent 函数已废弃，统一使用此函数以享受性能优化
func (m *Model) highlightCachedLines() string {
	var sb strings.Builder

	// 计算可见范围
	m.visibleStart = m.viewport.YOffset
	m.visibleEnd = m.visibleStart + m.viewport.Height
	lazyBuffer := 20 // 可见区域前后缓冲行数

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

		// 判断是否需要高亮（惰性高亮优化）
		needsHighlight := !m.highlightLazy ||
			(lineNum >= m.visibleStart-lazyBuffer &&
				lineNum <= m.visibleEnd+lazyBuffer)

		if !needsHighlight {
			// 不在可见范围，直接显示原文
			sb.WriteString(line)
			continue
		}

		// 检查缓存
		cacheKey := lineNum
		if cached, ok := m.highlightCache.Get(cacheKey); ok {
			// 使用缓存的高亮结果
			sb.WriteString(cached)
			continue
		}

		// 缓存未命中，需要高亮
		var highlighted string

		// JSON Formatting
		if m.prettyJson {
			// 尝试从行中提取JSON
			prefix, jsonPart, suffix, found := formatter.ExtractJSON(line)
			if found {
				// 格式化JSON部分
				formattedJSON, err := m.formatter.Format(jsonPart)
				if err == nil {
					// 成功格式化JSON
					var jsonBuilder strings.Builder
					if len(prefix) > 0 {
						// 对前缀应用高亮
						jsonBuilder.WriteString(m.highlighter.HighlightLine(prefix, lineNum))
					}
					jsonBuilder.WriteString(formattedJSON)
					if len(suffix) > 0 {
						jsonBuilder.WriteString(suffix)
					}
					highlighted = jsonBuilder.String()
					// 缓存并写入
					m.highlightCache.Add(cacheKey, highlighted)
					sb.WriteString(highlighted)
					continue
				}
			}
		}

		// 使用Chroma进行语法高亮
		highlighted = m.highlighter.HighlightLine(line, lineNum)

		// 如果启用搜索高亮且有搜索关键词，添加搜索匹配高亮
		if m.highlightMatches && m.searchQuery != "" {
			if strings.Contains(strings.ToLower(line), m.searchQuery) {
				highlighted = m.highlightSearchMatches(highlighted, m.searchQuery)
			}
		}

		// 缓存高亮结果
		m.highlightCache.Add(cacheKey, highlighted)

		// 应用软换行（如果启用）
		if m.softWrap && m.viewport.Width > 0 {
			// 计算可用宽度（减去行号宽度）
			availableWidth := m.viewport.Width
			if m.showLineNumbers {
				availableWidth -= 10 // 行号占用约10个字符
			}

			wrappedLines := m.wrapLine(highlighted, availableWidth)
			for j, wrappedLine := range wrappedLines {
				if j > 0 {
					sb.WriteString("\n")
					// 后续行添加缩进对齐
					if m.showLineNumbers {
						sb.WriteString("          ") // 10个空格对齐
					}
				}
				sb.WriteString(wrappedLine)
			}
		} else {
			sb.WriteString(highlighted)
		}
	}
	return sb.String()
}

// wrapLine 将长行按照指定宽度进行换行
// 在空格处断行，保持单词完整性
func (m *Model) wrapLine(line string, width int) []string {
	if width <= 0 {
		return []string{line}
	}

	// 移除ANSI转义序列计算可见长度
	visibleLen := lipgloss.Width(line)
	if visibleLen <= width {
		return []string{line}
	}

	// 简单实现：按空格分词
	words := strings.Fields(line)
	if len(words) == 0 {
		return []string{line}
	}

	var wrapped []string
	var currentLine strings.Builder
	currentWidth := 0

	for _, word := range words {
		wordWidth := lipgloss.Width(word)

		// 如果单词本身超过宽度，直接作为一行
		if wordWidth > width {
			if currentLine.Len() > 0 {
				wrapped = append(wrapped, currentLine.String())
				currentLine.Reset()
				currentWidth = 0
			}
			wrapped = append(wrapped, word)
			continue
		}

		// 计算添加这个词后的总宽度（包括空格）
		needWidth := wordWidth
		if currentWidth > 0 {
			needWidth += 1 // 空格
		}

		// 如果会超过宽度，换行
		if currentWidth+needWidth > width {
			wrapped = append(wrapped, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(word)
			currentWidth = wordWidth
		} else {
			// 添加到当前行
			if currentWidth > 0 {
				currentLine.WriteString(" ")
			}
			currentLine.WriteString(word)
			currentWidth += needWidth
		}
	}

	// 添加最后一行
	if currentLine.Len() > 0 {
		wrapped = append(wrapped, currentLine.String())
	}

	if len(wrapped) == 0 {
		return []string{line}
	}

	return wrapped
}

// highlightSearchMatches 为搜索匹配添加背景高亮
// 注意：text可能已经包含ANSI转义序列，需要小心处理
func (m *Model) highlightSearchMatches(text, query string) string {
	if query == "" {
		return text
	}

	matchStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("226")).
		Foreground(lipgloss.Color("0"))

	lowerText := strings.ToLower(text)
	lowerQuery := strings.ToLower(query)

	var builder strings.Builder
	lastIndex := 0
	searchStart := 0
	found := false

	for {
		pos := strings.Index(lowerText[searchStart:], lowerQuery)
		if pos == -1 {
			break
		}

		found = true
		matchStart := searchStart + pos
		matchEnd := matchStart + len(query)
		builder.WriteString(text[lastIndex:matchStart])
		builder.WriteString(matchStyle.Render(text[matchStart:matchEnd]))
		lastIndex = matchEnd
		searchStart = matchEnd
	}

	if !found {
		return text
	}

	builder.WriteString(text[lastIndex:])
	return builder.String()
}

// shouldRender 检查是否需要重新渲染viewport内容
// 通过比较内容hash来避免不必要的SetContent调用
func (m *Model) shouldRender(newHash uint64) bool {
	if m.needsFullRender || newHash != m.lastRenderedHash {
		m.lastRenderedHash = newHash
		m.needsFullRender = false
		return true
	}
	return false
}

// View 渲染日志。
func (m Model) View() string {
	if m.err != nil {
		return m.panel.RenderEmpty(fmt.Sprintf("无法加载日志:\n%v", m.err))
	}
	if m.loading {
		return m.panel.RenderEmpty("正在加载日志，请稍候...\n(大文件正在建立索引)")
	}
	if m.history == nil && m.filePath == "" {
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
		m.searchQuery = "" // 清除搜索关键词
		return
	}

	lowerQuery := strings.ToLower(query)
	m.searchQuery = lowerQuery // 保存搜索关键词用于高亮
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

	// 大文件模式：优先使用ripgrep快速搜索
	if m.usesChunked && m.chunkedReader != nil {
		var filePath string
		if m.filePath != "" {
			filePath = m.filePath
		} else if m.history != nil {
			filePath = m.history.LogFilePath
		}

		// 尝试使用ripgrep
		if filePath != "" && reader.RipgrepAvailable() {
			matches, err := reader.RipgrepSearch(filePath, query)
			if err == nil {
				m.searchMatches = matches

				if len(m.searchMatches) > 0 {
					// 找到从当前位置开始的第一个匹配
					currentOffset := m.viewport.YOffset
					for i, lineIdx := range m.searchMatches {
						if lineIdx >= currentOffset {
							m.currentMatch = i
							m.viewport.SetYOffset(lineIdx)
							m.statusMsg = fmt.Sprintf("找到 %d 个匹配，当前 %d/%d (ripgrep)", len(m.searchMatches), i+1, len(m.searchMatches))
							return
						}
					}
					// 如果所有匹配都在当前位置之前，跳转到第一个匹配
					m.currentMatch = 0
					m.viewport.SetYOffset(m.searchMatches[0])
					m.statusMsg = fmt.Sprintf("找到 %d 个匹配，当前 1/%d (ripgrep)", len(m.searchMatches), len(m.searchMatches))
				} else {
					m.statusMsg = "未找到匹配 (ripgrep)"
				}
				return
			}
			// ripgrep失败，继续使用fallback方法
		}

		// Fallback: 使用原有的扫描方法
		matches, err := reader.FallbackSearch(m.chunkedReader, query)
		if err != nil {
			m.statusMsg = fmt.Sprintf("搜索出错: %v", err)
			return
		}

		m.searchMatches = matches

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
