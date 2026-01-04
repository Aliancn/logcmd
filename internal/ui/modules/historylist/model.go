package historylist

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aliancn/logcmd/internal/history"
	"github.com/aliancn/logcmd/internal/model"
	"github.com/aliancn/logcmd/internal/ui/common"
)

// Model 展示项目运行历史。
type Model struct {
	manager *history.Manager
	project *model.Project
	list    list.Model
	keys    keyMap
	styles  common.Styles
	width   int
	height  int
	filter  statusFilter
}

// HistoriesLoadedMsg 表示历史记录加载完成。
type HistoriesLoadedMsg struct {
	ProjectID int
	Histories []*model.CommandHistory
}

// BackToProjectsMsg 请求返回项目列表。
type BackToProjectsMsg struct{}

// OpenLogMsg 请求打开日志。
type OpenLogMsg struct {
	History *model.CommandHistory
}

// New 创建历史列表 Model。
func New(manager *history.Manager) Model {
	styles := common.DefaultStyles()
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true

	l := list.New(nil, delegate, 0, 0)
	l.Title = "历史记录"
	l.Styles.Title = styles.Title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)

	keys := newKeyMap()
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keys.Open, keys.Refresh, keys.Filter}
	}

	model := Model{
		manager: manager,
		list:    l,
		keys:    keys,
		styles:  styles,
		filter:  filterAll,
	}
	model.updateTitle()
	return model
}

// SetProject 设定当前项目。
func (m *Model) SetProject(project *model.Project) {
	m.project = project
	m.list.SetItems(nil)
	m.updateTitle()
}

// Init 实现 tea.Model，需要与父 model 一致。
func (m Model) Init() tea.Cmd {
	return nil
}

// SetSize 调整组件大小。
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	w := width - 4
	h := height - 4
	if w < 30 {
		w = width
	}
	if h < 5 {
		h = height
	}
	m.list.SetSize(w, h)
}

// LoadHistoryCmd 触发历史加载。
func (m Model) LoadHistoryCmd() tea.Cmd {
	if m.project == nil || m.manager == nil {
		return nil
	}
	projectID := m.project.ID
	status := m.filter.statusValue()
	return func() tea.Msg {
		items, err := m.manager.Query(history.QueryOptions{
			ProjectID: projectID,
			Status:    status,
			Limit:     200,
		})
		if err != nil {
			return common.ErrorMsg{Err: fmt.Errorf("加载历史失败: %w", err)}
		}
		return HistoriesLoadedMsg{
			ProjectID: projectID,
			Histories: items,
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
		switch {
		case key.Matches(msg, m.keys.Back):
			cmds = append(cmds, func() tea.Msg { return BackToProjectsMsg{} })
		case key.Matches(msg, m.keys.Open):
			if item, ok := m.list.SelectedItem().(historyItem); ok {
				h := item.history
				cmds = append(cmds, func() tea.Msg { return OpenLogMsg{History: h} })
			}
		case key.Matches(msg, m.keys.Refresh):
			cmds = append(cmds, m.LoadHistoryCmd())
		case key.Matches(msg, m.keys.Filter):
			m.filter = m.filter.next()
			m.updateTitle()
			cmds = append(cmds, m.LoadHistoryCmd())
		}
	case HistoriesLoadedMsg:
		if m.project == nil || msg.ProjectID != m.project.ID {
			break
		}
		items := make([]list.Item, len(msg.Histories))
		for i, history := range msg.Histories {
			items[i] = historyItem{history: history}
		}
		m.list.SetItems(items)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View 渲染列表。
func (m Model) View() string {
	if m.project == nil {
		return m.styles.Frame.Render("在左侧选择一个项目以查看历史记录")
	}
	return m.styles.Frame.Render(m.list.View())
}

type historyItem struct {
	history *model.CommandHistory
}

func (i historyItem) Title() string {
	command := strings.TrimSpace(i.history.CommandName)
	if command == "" {
		command = i.history.Command
	}
	start := i.history.StartTime.Format("2006-01-02 15:04:05")
	duration := time.Duration(i.history.DurationMs) * time.Millisecond
	statusLabel := strings.ToUpper(i.history.Status)
	exit := fmt.Sprintf("%3d", i.history.ExitCode)
	return fmt.Sprintf("%s %-7s %s  退出:%s  耗时:%-7s  %s",
		statusSymbol(i.history.Status),
		statusLabel,
		start,
		exit,
		formatDuration(duration),
		command)
}

func (i historyItem) Description() string {
	logPath := i.history.LogFilePath
	if logPath == "" {
		logPath = "-"
	}
	workdir := i.history.WorkingDirectory
	if workdir == "" {
		workdir = "-"
	}
	return fmt.Sprintf("目录: %s | 日志: %s", workdir, logPath)
}

func (i historyItem) FilterValue() string {
	return i.history.Command
}

func statusSymbol(status string) string {
	switch status {
	case "success":
		return "✅"
	case "failed":
		return "❌"
	default:
		return "•"
	}
}

type keyMap struct {
	Back    key.Binding
	Open    key.Binding
	Refresh key.Binding
	Filter  key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "返回项目"),
		),
		Open: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "查看日志"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "刷新"),
		),
		Filter: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "切换状态筛选"),
		),
	}
}

// CurrentProjectID 返回当前项目 ID。
func (m Model) CurrentProjectID() int {
	if m.project == nil {
		return 0
	}
	return m.project.ID
}

func (m *Model) updateTitle() {
	filterLabel := m.filter.label()
	if m.project != nil {
		m.list.Title = fmt.Sprintf("项目 #%d 历史记录 | 筛选: %s", m.project.ID, filterLabel)
		return
	}
	m.list.Title = fmt.Sprintf("历史记录 | 筛选: %s", filterLabel)
}

type statusFilter int

const (
	filterAll statusFilter = iota
	filterSuccess
	filterFailed
)

func (f statusFilter) next() statusFilter {
	switch f {
	case filterAll:
		return filterSuccess
	case filterSuccess:
		return filterFailed
	default:
		return filterAll
	}
}

func (f statusFilter) label() string {
	switch f {
	case filterSuccess:
		return "成功"
	case filterFailed:
		return "失败"
	default:
		return "全部"
	}
}

func (f statusFilter) statusValue() string {
	switch f {
	case filterSuccess:
		return "success"
	case filterFailed:
		return "failed"
	default:
		return ""
	}
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "-"
	}
	if d >= time.Hour {
		hours := int(d / time.Hour)
		mins := int((d % time.Hour) / time.Minute)
		return fmt.Sprintf("%dh%02dm", hours, mins)
	}
	if d >= time.Minute {
		mins := int(d / time.Minute)
		secs := int((d % time.Minute) / time.Second)
		return fmt.Sprintf("%dm%02ds", mins, secs)
	}
	if d >= time.Second {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%dms", d.Milliseconds())
}
