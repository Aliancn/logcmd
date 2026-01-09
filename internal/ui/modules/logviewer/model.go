package logviewer

import (
	"bytes"
	"encoding/json"
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
)

// Model 展示日志内容。
type Model struct {
	viewport    viewport.Model
	panel       *panel.Panel
	history     *model.CommandHistory
	content     string
	width       int
	height      int
	theme       common.Theme
	styles      common.Styles
	keys        keyMap
	searchInput textinput.Model
	searching   bool
	lastQuery   string
	statusMsg   string
	prettyJson  bool // JSON 格式化模式
}

// ContentLoadedMsg 表示日志内容已加载。
type ContentLoadedMsg struct {
	HistoryID int
	Content   string
}

// BackMsg 表示用户请求返回上一视图。
type BackMsg struct{}

// New 创建日志查看器。
func New(theme common.Theme, styles common.Styles) Model {
	vp := viewport.New(0, 0)
	input := textinput.New()
	input.Prompt = "/ "
	input.Placeholder = "输入搜索关键词"

	// 创建Panel布局容器
	p := panel.NewDefault("", theme, styles)

	return Model{
		viewport:    vp,
		panel:       p,
		theme:       theme,
		styles:      styles,
		keys:        newKeyMap(),
		searchInput: input,
	}
}

// Init 实现 tea.Model。
func (m Model) Init() tea.Cmd {
	return nil
}

// SetHistory 指定要查看的历史记录。
func (m *Model) SetHistory(history *model.CommandHistory) {
	m.history = history
	m.content = ""
	m.viewport.SetContent("")
	m.viewport.GotoTop()
	m.statusMsg = ""
}

// Reset 清理状态。
func (m *Model) Reset() {
	m.history = nil
	m.content = ""
	m.viewport.SetContent("")
	m.statusMsg = ""
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
	}

	// 构建footer
	var footer string
	if m.searching {
		footer = m.searchInput.View()
	} else {
		defaultHints := "j/k 滚动 · gg/G 跳转 · / 搜索"
		if m.statusMsg != "" {
			footer = common.JoinKeyHelps(m.statusMsg, defaultHints)
		} else {
			footer = defaultHints
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
	if m.history == nil || m.history.LogFilePath == "" {
		return nil
	}
	historyID := m.history.ID
	path := m.history.LogFilePath
	return func() tea.Msg {
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
			} else if msg.Type == tea.KeyEsc {
				m.searching = false
				m.searchInput.Blur()
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
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keys.ToggleJSON):
			m.prettyJson = !m.prettyJson
			// Re-render content
			highlighted := m.highlightContent(m.content)
			m.viewport.SetContent(highlighted)
			status := "JSON格式化: 关闭"
			if m.prettyJson {
				status = "JSON格式化: 开启"
			}
			m.statusMsg = status
		case key.Matches(msg, m.keys.Down):
			m.viewport.LineDown(1)
		case key.Matches(msg, m.keys.Up):
			m.viewport.LineUp(1)
		case key.Matches(msg, m.keys.PageDown):
			m.viewport.ViewDown()
		case key.Matches(msg, m.keys.PageUp):
			m.viewport.ViewUp()
		case key.Matches(msg, m.keys.Top):
			m.viewport.GotoTop()
		case key.Matches(msg, m.keys.Bottom):
			m.viewport.GotoBottom()
		}
	case ContentLoadedMsg:
		if m.history == nil || msg.HistoryID != m.history.ID {
			break
		}
		m.content = msg.Content // Keep raw content for search/logic

		// Apply Highlighting
		highlighted := m.highlightContent(msg.Content)

		m.viewport.SetContent(highlighted)
		m.viewport.GotoTop()
		m.statusMsg = fmt.Sprintf("日志加载完成（%d 字节）", len(msg.Content))
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

	for i, line := range lines {
		if i > 0 {
			sb.WriteString("\n")
		}

		// JSON Formatting
		if m.prettyJson {
			// Find first '{' and last '}'
			start := strings.Index(line, "{")
			end := strings.LastIndex(line, "}")

			if start >= 0 && end > start {
				jsonPart := line[start : end+1]
				var buf bytes.Buffer
				if err := json.Indent(&buf, []byte(jsonPart), "", "  "); err == nil {
					// Valid JSON, replace part
					formatted := buf.String()
					// Indent subsequent lines of JSON to match start pos?
					// For simplicity, just inject it.
					// We colorize the JSON part? Maybe later.
					prefix := line[:start]
					suffix := line[end+1:]

					// Re-assemble
					// Note: this makes line multiline. highlight logic below works on line-by-line basis
					// but we are inside a loop over original lines.
					// We need to append the formatted block.

					// Style the prefix if needed
					if len(prefix) > 0 {
						sb.WriteString(m.styleLine(prefix)) // Recursive style? No, separate helper.
					}

					// Append formatted JSON (lipgloss doesn't indent multiline automatically well without help)
					sb.WriteString(formatted)

					if len(suffix) > 0 {
						sb.WriteString(suffix)
					}
					continue // Skip standard processing for this line
				}
			}
		}

		// Standard Highlighting logic (extracted from previous code)
		sb.WriteString(m.styleLine(line))
	}
	return sb.String()
}

// styleLine applies standard regex/keyword highlighting
func (m Model) styleLine(line string) string {
	// Define styles (reused from before, but need to be accessible)
	// I'll re-instantiate or move them to struct. For now re-instantiate is cheap.
	timeStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)
	debugStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)
	infoStyle := lipgloss.NewStyle().Foreground(m.theme.Primary)
	warnStyle := lipgloss.NewStyle().Foreground(m.theme.Warning)
	errorStyle := lipgloss.NewStyle().Foreground(m.theme.Error).Bold(true)
	fatalStyle := lipgloss.NewStyle().Background(m.theme.Error).Foreground(lipgloss.Color("#FFFFFF")).Bold(true)

	styledLine := line
	lowerLine := strings.ToLower(line)

	// 1. Highlight Timestamp
	if len(line) > 20 {
		firstSpace := strings.Index(line, " ")
		if firstSpace > 5 && firstSpace < 30 {
			prefix := line[:firstSpace]
			if strings.ContainsAny(prefix, "0123456789") {
				if !strings.Contains(lowerLine, "fatal") && !strings.Contains(lowerLine, "error") && !strings.Contains(lowerLine, "err]") && !strings.Contains(lowerLine, "warn") {
					styledLine = timeStyle.Render(prefix) + styledLine[firstSpace:]
				}
			}
		}
	}

	// 2. Highlight Level
	switch {
	case strings.Contains(lowerLine, "fatal"):
		styledLine = fatalStyle.Render(line)
	case strings.Contains(lowerLine, "error") || strings.Contains(lowerLine, "err]"):
		styledLine = errorStyle.Render(line)
	case strings.Contains(lowerLine, "warn"):
		styledLine = warnStyle.Render(line)
	case strings.Contains(lowerLine, "info"):
		styledLine = strings.Replace(styledLine, "INFO", infoStyle.Render("INFO"), 1)
		styledLine = strings.Replace(styledLine, "info", infoStyle.Render("info"), 1)
	case strings.Contains(lowerLine, "debug"):
		styledLine = debugStyle.Render(line)
	}

	return styledLine
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

// jumpToQuery 在日志中查找文本。
func (m *Model) jumpToQuery(query string) {
	if query == "" || m.content == "" {
		m.statusMsg = "未输入搜索关键词"
		return
	}

	lines := strings.Split(m.content, "\n")
	start := m.viewport.YOffset
	start++
	lowerQuery := strings.ToLower(query)
	for idx := 0; idx < len(lines); idx++ {
		lineIdx := (start + idx) % len(lines)
		line := strings.ToLower(lines[lineIdx])
		if strings.Contains(line, lowerQuery) {
			m.viewport.SetYOffset(lineIdx)
			m.statusMsg = fmt.Sprintf("匹配行: %d", lineIdx+1)
			return
		}
	}
	m.statusMsg = "未找到匹配"
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

type keyMap struct {
	Back       key.Binding
	Search     key.Binding
	Down       key.Binding
	Up         key.Binding
	PageDown   key.Binding
	PageUp     key.Binding
	Top        key.Binding
	Bottom     key.Binding
	ToggleJSON key.Binding
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
		ToggleJSON: key.NewBinding(
			key.WithKeys("ctrl+j"),
			key.WithHelp("ctrl+j", "JSON格式化"),
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
