package historylist

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aliancn/logcmd/internal/platform/history"
	"github.com/aliancn/logcmd/internal/domain/model"
	"github.com/aliancn/logcmd/internal/presentation/ui/common"
	"github.com/aliancn/logcmd/internal/presentation/ui/components/panel"
)

// Model 展示项目运行历史。
type Model struct {
	manager          *history.Manager
	project          *model.Project
	list             list.Model
	panel            *panel.Panel
	keys             keyMap
	theme            common.Theme
	styles           common.Styles
	width            int
	height           int
	filter           statusFilter
	statusMsg        string
	confirmingDelete bool
	deleteTargetID   int
	deleteTargetName string
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

// HistoryDeletedMsg 表示删除完成。
type HistoryDeletedMsg struct {
	HistoryID int
}

// New 创建历史列表 Model。
func New(manager *history.Manager, theme common.Theme, styles common.Styles) Model {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true

	l := list.New(nil, delegate, 0, 0)
	// 禁用列表内置退出键，避免 Esc 被视作全局退出
	l.DisableQuitKeybindings()
	l.Title = "历史记录"
	l.Styles.Title = styles.Title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)

	keys := newKeyMap()
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keys.Open, keys.Refresh, keys.Filter, keys.Delete}
	}

	// 创建Panel布局容器
	p := panel.NewDefault("", theme, styles)

	model := Model{
		manager: manager,
		list:    l,
		panel:   p,
		keys:    keys,
		theme:   theme,
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
	m.resetDeleteState()
	m.setStatus("")
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

	// 设置Panel尺寸，Panel会自动计算内容区域
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

	// 设置list使用精确的内容尺寸
	m.list.SetSize(contentW, contentH)

	m.updateFooter()
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
	shouldUpdateList := true

	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(typed.Width, typed.Height)
	case tea.KeyMsg:
		if m.confirmingDelete {
			shouldUpdateList = false
			var cmd tea.Cmd
			m, cmd = m.handleDeleteConfirmation(typed)
			cmds = append(cmds, cmd)
			break
		}
		switch {
		case key.Matches(typed, m.keys.Open):
			if item, ok := m.list.SelectedItem().(historyItem); ok {
				h := item.history
				cmds = append(cmds, func() tea.Msg { return OpenLogMsg{History: h} })
			}
		case key.Matches(typed, m.keys.Refresh):
			cmds = append(cmds, m.LoadHistoryCmd())
		case key.Matches(typed, m.keys.Filter):
			m.filter = m.filter.next()
			m.updateTitle()
			cmds = append(cmds, m.LoadHistoryCmd())
		case key.Matches(typed, m.keys.Delete):
			shouldUpdateList = false
			if item, ok := m.list.SelectedItem().(historyItem); ok && item.history != nil {
				m.confirmingDelete = true
				m.deleteTargetID = item.history.ID
				name := strings.TrimSpace(item.history.CommandName)
				if name == "" {
					name = strings.TrimSpace(item.history.Command)
				}
				m.deleteTargetName = name
				m.updateFooter()
			} else {
				m.setStatus("请选择一条记录")
			}
		}
	case HistoriesLoadedMsg:
		if m.project == nil || typed.ProjectID != m.project.ID {
			break
		}
		items := make([]list.Item, len(typed.Histories))
		for i, history := range typed.Histories {
			items[i] = historyItem{history: history}
		}
		m.list.SetItems(items)
		m.setStatus(fmt.Sprintf("共 %d 条记录", len(items)))
	case HistoryDeletedMsg:
		m.setStatus(fmt.Sprintf("记录 #%d 已删除", typed.HistoryID))
		cmds = append(cmds, m.LoadHistoryCmd())
	}

	if shouldUpdateList {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View 渲染列表。
func (m Model) View() string {
	if m.project == nil {
		return m.panel.RenderEmpty("在左侧选择一个项目以查看历史记录")
	}
	return m.panel.Render(m.list.View())
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
	Open    key.Binding
	Refresh key.Binding
	Filter  key.Binding
	Delete  key.Binding
	Confirm key.Binding
	Cancel  key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
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
		Delete: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "删除记录"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("y", "Y", "enter"),
			key.WithHelp("y/enter", "确认删除"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("n", "N", "esc"),
			key.WithHelp("n/esc", "取消"),
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

func (m Model) handleDeleteConfirmation(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Confirm):
		targetID := m.deleteTargetID
		if targetID <= 0 {
			m.resetDeleteState()
			m.setStatus("未找到要删除的记录")
			return m, nil
		}
		m.resetDeleteState()
		m.setStatus(fmt.Sprintf("正在删除记录 #%d...", targetID))
		return m, m.deleteHistoryCmd(targetID)
	case key.Matches(msg, m.keys.Cancel):
		m.resetDeleteState()
		m.setStatus("已取消删除")
	}
	return m, nil
}

func (m Model) deleteHistoryCmd(historyID int) tea.Cmd {
	if historyID <= 0 {
		return nil
	}
	if m.manager == nil {
		return func() tea.Msg {
			return common.ErrorMsg{Err: fmt.Errorf("无法删除记录: 历史管理器未初始化")}
		}
	}
	return func() tea.Msg {
		if err := m.manager.Delete(historyID); err != nil {
			return common.ErrorMsg{Err: fmt.Errorf("删除历史记录失败: %w", err)}
		}
		return HistoryDeletedMsg{HistoryID: historyID}
	}
}

func (m *Model) setStatus(msg string) {
	m.statusMsg = msg
	m.updateFooter()
}

func (m *Model) updateFooter() {
	if m.panel == nil {
		return
	}

	defaultHints := common.JoinKeyHelps(
		common.FormatKeyHelp(m.keys.Open),
		common.FormatKeyHelp(m.keys.Refresh),
		common.FormatKeyHelp(m.keys.Filter),
		common.FormatKeyHelp(m.keys.Delete),
	)

	footer := defaultHints
	if m.confirmingDelete {
		label := fmt.Sprintf("#%d", m.deleteTargetID)
		name := strings.TrimSpace(m.deleteTargetName)
		if name != "" {
			label = fmt.Sprintf("%s %s", label, name)
		}
		footer = common.JoinKeyHelps(
			fmt.Sprintf("确认删除 %s", label),
			common.FormatKeyHelp(m.keys.Confirm),
			common.FormatKeyHelp(m.keys.Cancel),
		)
	} else if strings.TrimSpace(m.statusMsg) != "" {
		footer = common.JoinKeyHelps(m.statusMsg, defaultHints)
	}

	m.panel.SetFooter(footer)
}

func (m *Model) resetDeleteState() {
	m.confirmingDelete = false
	m.deleteTargetID = 0
	m.deleteTargetName = ""
}
