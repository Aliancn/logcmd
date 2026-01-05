package taskmanager

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aliancn/logcmd/internal/model"
	"github.com/aliancn/logcmd/internal/tasks"
	"github.com/aliancn/logcmd/internal/tasks/operations"
	"github.com/aliancn/logcmd/internal/ui/common"
	"github.com/aliancn/logcmd/internal/ui/components/panel"
)

// Model 展示后台任务列表。
type Model struct {
	manager          *tasks.Manager
	list             list.Model
	panel            *panel.Panel
	keys             keyMap
	theme            common.Theme
	styles           common.Styles
	width            int
	height           int
	statusMsg        string
	refreshInterval  time.Duration
	active           bool
	lastRefreshError error
}

// tasksLoadedMsg 表示任务列表加载完成。
type tasksLoadedMsg struct {
	tasks []*model.Task
}

type refreshTickMsg struct{}

type taskActionMsg struct {
	taskID int
	action string
}

// New 创建任务管理模块。
func New(manager *tasks.Manager, theme common.Theme, styles common.Styles) Model {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true

	l := list.New(nil, delegate, 0, 0)
	l.Title = "后台任务"
	l.Styles.Title = styles.Title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	// 使用Panel自定义footer展示快捷键及状态，避免list默认帮助重复
	l.SetShowHelp(false)

	keys := newKeyMap()
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keys.Refresh, keys.Stop, keys.Kill}
	}

	// 创建Panel布局容器
	p := panel.NewDefault("", theme, styles)

	return Model{
		manager:         manager,
		list:            l,
		panel:           p,
		keys:            keys,
		theme:           theme,
		styles:          styles,
		refreshInterval: 5 * time.Second,
	}
}

// Init 实现 tea.Model。
func (m Model) Init() tea.Cmd {
	return nil
}

// SetSize 调整列表尺寸。
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height

	// 更新footer（如果有状态消息则显示，否则显示帮助）
	var footer string
	if m.statusMsg != "" {
		footer = m.styles.StatusBar.Render(m.statusMsg)
	} else {
		footer = m.styles.StatusBar.Render("r 刷新 · s 停止 · k 终止 · tab 返回")
	}
	m.panel.SetFooter(footer)

	// 设置Panel尺寸，Panel会自动计算内容区域（已扣除footer）
	m.panel.SetSize(width, height)

	// 获取Panel计算后的精确内容尺寸
	contentW, contentH := m.panel.GetContentSize()

	// 确保最小尺寸
	if contentW < 40 {
		contentW = 40
	}
	if contentH < 5 {
		contentH = 5
	}

	// 设置list使用精确的内容尺寸
	m.list.SetSize(contentW, contentH)
}

// SetActive 控制刷新生命周期。
func (m *Model) SetActive(active bool) tea.Cmd {
	m.active = active
	if !active {
		return nil
	}
	return tea.Batch(m.loadTasksCmd(), m.tickCmd())
}

// Update 处理任务管理消息。
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Refresh):
			cmds = append(cmds, m.loadTasksCmd())
		case key.Matches(msg, m.keys.Stop):
			if task := m.selectedTask(); task != nil {
				cmds = append(cmds, m.stopTaskCmd(task, false))
			}
		case key.Matches(msg, m.keys.Kill):
			if task := m.selectedTask(); task != nil {
				cmds = append(cmds, m.stopTaskCmd(task, true))
			}
		}
	case refreshTickMsg:
		if m.active {
			cmds = append(cmds, tea.Batch(m.loadTasksCmd(), m.tickCmd()))
		}
	case tasksLoadedMsg:
		items := make([]list.Item, 0, len(msg.tasks))
		for _, task := range msg.tasks {
			items = append(items, newTaskItem(task))
		}
		m.list.SetItems(items)
		m.lastRefreshError = nil
		if len(items) == 0 {
			m.statusMsg = "暂无运行中的任务"
		} else {
			m.statusMsg = fmt.Sprintf("已刷新 %d 个任务", len(items))
		}
	case taskActionMsg:
		m.statusMsg = fmt.Sprintf("任务 #%d 已%s", msg.taskID, msg.action)
		cmds = append(cmds, m.loadTasksCmd())
	case common.ErrorMsg:
		m.lastRefreshError = msg.Err
		m.statusMsg = fmt.Sprintf("错误: %v", msg.Err)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View 渲染任务列表。
func (m Model) View() string {
	// 确保footer是最新的
	var footer string
	if m.statusMsg != "" {
		footer = m.styles.StatusBar.Render(m.statusMsg)
	} else {
		footer = m.styles.StatusBar.Render("r 刷新 · s 停止 · k 终止 · tab 返回")
	}
	m.panel.SetFooter(footer)

	// 使用Panel渲染list内容
	return m.panel.Render(m.list.View())
}

func (m Model) selectedTask() *model.Task {
	item, ok := m.list.SelectedItem().(taskItem)
	if !ok {
		return nil
	}
	return item.task
}

func (m Model) loadTasksCmd() tea.Cmd {
	if m.manager == nil {
		return nil
	}
	return func() tea.Msg {
		tasksList, err := m.manager.ListActive()
		if err != nil {
			return common.ErrorMsg{Err: fmt.Errorf("读取任务失败: %w", err)}
		}
		return tasksLoadedMsg{tasks: tasksList}
	}
}

func (m Model) stopTaskCmd(task *model.Task, force bool) tea.Cmd {
	if m.manager == nil || task == nil {
		return nil
	}
	taskID := task.ID
	return func() tea.Msg {
		action, err := operations.StopTask(m.manager, task, force)
		if err != nil {
			return common.ErrorMsg{Err: err}
		}
		return taskActionMsg{
			taskID: taskID,
			action: action,
		}
	}
}

func (m Model) tickCmd() tea.Cmd {
	if m.refreshInterval <= 0 {
		return nil
	}
	interval := m.refreshInterval
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return refreshTickMsg{}
	})
}

type taskItem struct {
	task  *model.Task
	alive bool
}

func newTaskItem(task *model.Task) taskItem {
	alive := operations.CheckProcessAlive(task.PID)
	return taskItem{
		task:  task,
		alive: alive,
	}
}

func (i taskItem) Title() string {
	status := i.task.Status
	if i.alive {
		status = "alive/" + i.task.Status
	} else if i.task.IsActive() {
		status = "lost"
	}
	command := formatTaskCommand(i.task)
	return fmt.Sprintf("#%d [%s] %s", i.task.ID, status, command)
}

func (i taskItem) Description() string {
	started := "-"
	if i.task.StartedAt != nil {
		started = i.task.StartedAt.Format("2006-01-02 15:04:05")
	}
	pid := "-"
	if i.task.PID != nil {
		pid = fmt.Sprintf("%d", *i.task.PID)
	}
	var parts []string
	parts = append(parts, fmt.Sprintf("PID:%s", pid))
	parts = append(parts, fmt.Sprintf("创建:%s", i.task.CreatedAt.Format("2006-01-02 15:04:05")))
	parts = append(parts, fmt.Sprintf("开始:%s", started))
	if i.task.LogFilePath != "" {
		parts = append(parts, fmt.Sprintf("日志:%s", i.task.LogFilePath))
	}
	return strings.Join(parts, " | ")
}

func (i taskItem) FilterValue() string {
	return formatTaskCommand(i.task)
}

func formatTaskCommand(task *model.Task) string {
	return strings.Join(append([]string{task.Command}, task.CommandArgs...), " ")
}

type keyMap struct {
	Refresh key.Binding
	Stop    key.Binding
	Kill    key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "刷新"),
		),
		Stop: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "停止任务"),
		),
		Kill: key.NewBinding(
			key.WithKeys("k"),
			key.WithHelp("k", "终止任务"),
		),
	}
}
