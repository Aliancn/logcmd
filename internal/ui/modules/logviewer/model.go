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

	"github.com/aliancn/logcmd/internal/model"
	"github.com/aliancn/logcmd/internal/ui/common"
)

// Model 展示日志内容。
type Model struct {
	viewport    viewport.Model
	history     *model.CommandHistory
	content     string
	width       int
	height      int
	styles      common.Styles
	keys        keyMap
	searchInput textinput.Model
	searching   bool
	lastQuery   string
	statusMsg   string
}

// ContentLoadedMsg 表示日志内容已加载。
type ContentLoadedMsg struct {
	HistoryID int
	Content   string
}

// BackMsg 表示用户请求返回上一视图。
type BackMsg struct{}

// New 创建日志查看器。
func New() Model {
	styles := common.DefaultStyles()
	vp := viewport.New(0, 0)
	input := textinput.New()
	input.Prompt = "/ "
	input.Placeholder = "输入搜索关键词"

	return Model{
		viewport:    vp,
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
	w := width - 4
	h := height - 6
	if w < 30 {
		w = width
	}
	if h < 5 {
		h = height - 2
	}
	m.viewport.Width = w
	m.viewport.Height = h
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
		m.content = msg.Content
		m.viewport.SetContent(msg.Content)
		m.viewport.GotoTop()
		m.statusMsg = fmt.Sprintf("日志加载完成（%d 字节）", len(msg.Content))
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View 渲染日志。
func (m Model) View() string {
	if m.history == nil {
		return m.styles.Frame.Render("选择一条历史记录查看日志")
	}

	header := fmt.Sprintf("#%d %s\n日志文件: %s\n开始时间: %s\n",
		m.history.ID,
		m.history.Command,
		m.history.LogFilePath,
		m.history.StartTime.Format(time.RFC3339),
	)

	body := m.viewport.View()

	var footer string
	if m.searching {
		footer = m.searchInput.View()
	} else if m.statusMsg != "" {
		footer = m.styles.StatusBar.Render(m.statusMsg)
	} else {
		footer = m.styles.StatusBar.Render("j/k 滚动 · gg/G 跳转 · / 搜索")
	}

	view := strings.Builder{}
	view.WriteString(header)
	view.WriteString("\n")
	view.WriteString(body)
	view.WriteString("\n")
	view.WriteString(footer)

	return m.styles.Frame.Render(view.String())
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

type keyMap struct {
	Back     key.Binding
	Search   key.Binding
	Down     key.Binding
	Up       key.Binding
	PageDown key.Binding
	PageUp   key.Binding
	Top      key.Binding
	Bottom   key.Binding
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
