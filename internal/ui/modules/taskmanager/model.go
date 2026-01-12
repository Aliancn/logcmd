package taskmanager

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
	viewport         viewport.Model // Log preview
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

	// State for split view
	selectedID        int
	logPreviewContent string
}

// tasksLoadedMsg 表示任务列表加载完成。
type tasksLoadedMsg struct {
	tasks []*model.Task
}

type logPreviewMsg struct {
	taskID  int
	content string
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
	delegate.Styles.SelectedTitle = styles.ListItemSelected
	delegate.Styles.SelectedDesc = styles.ListItemSelected.Copy().Foreground(theme.TextMuted)

	l := list.New(nil, delegate, 0, 0)
	// 禁用默认退出键，统一由上层处理 Esc/Quit
	l.DisableQuitKeybindings()
	l.Title = "后台任务"
	l.SetShowTitle(false) // Custom header used
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	// 使用Panel自定义footer展示快捷键及状态，避免list默认帮助重复
	l.SetShowHelp(false)

	keys := newKeyMap()
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keys.Refresh, keys.Stop, keys.Kill}
	}

	vp := viewport.New(0, 0)

	// 创建Panel布局容器
	p := panel.NewDefault("", theme, styles)

	return Model{
		manager:         manager,
		list:            l,
		viewport:        vp,
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
	defaultHints := common.JoinKeyHelps(
		common.FormatKeyHelp(m.keys.Refresh),
		common.FormatKeyHelp(m.keys.Stop),
		common.FormatKeyHelp(m.keys.Kill),
	)
	if m.statusMsg != "" {
		footer = common.JoinKeyHelps(m.statusMsg, defaultHints)
	} else {
		footer = defaultHints
	}
	m.panel.SetFooter(footer)

	// Custom Header
	m.panel.SetHeader(lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render("Process Manager"))

	// 设置Panel尺寸，Panel会自动计算内容区域（已扣除footer）
	m.panel.SetSize(width, height)

	// 获取Panel计算后的精确内容尺寸
	contentW, contentH := m.panel.GetContentSize()

	if contentW < 40 {
		contentW = 40
	}
	if contentH < 5 {
		contentH = 5
	}

	// Split View Calculation
	// List takes 40% or min 30 chars
	listW := int(float64(contentW) * 0.4)
	if listW < 30 {
		listW = 30
	}
	// If screen is too small, fallback to list only (simplified responsiveness)
	if contentW < 60 {
		listW = contentW
	}

	detailW := contentW - listW - 2 // 2 chars for gap/border
	if detailW < 0 {
		detailW = 0
	}

	// 设置list使用精确的内容尺寸
	m.list.SetSize(listW, contentH)

	// Setup Viewport for log preview (bottom part of details)
	// Header ~ 2 lines, Meta ~ 6 lines, Gap ~ 1
	// Log Preview Height = contentH - 9
	vpH := contentH - 10
	if vpH < 5 {
		vpH = 5
	}
	m.viewport.Width = detailW
	m.viewport.Height = vpH
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

		// Check if we need to reload preview for current selection
		if task := m.selectedTask(); task != nil {
			if task.ID != m.selectedID {
				m.selectedID = task.ID
				cmds = append(cmds, m.loadLogPreviewCmd(task))
			}
		}

	case taskActionMsg:
		m.statusMsg = fmt.Sprintf("任务 #%d 已%s", msg.taskID, msg.action)
		cmds = append(cmds, m.loadTasksCmd())

	case logPreviewMsg:
		if m.selectedTask() != nil && m.selectedTask().ID == msg.taskID {
			m.logPreviewContent = msg.content
			m.viewport.SetContent(msg.content)
			m.viewport.GotoBottom()
		}

	case common.ErrorMsg:
		m.lastRefreshError = msg.Err
		m.statusMsg = fmt.Sprintf("错误: %v", msg.Err)
	}

	var cmd tea.Cmd

	// Check for selection change during list update
	prevSel := m.selectedTask()
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)
	newSel := m.selectedTask()

	if newSel != nil && (prevSel == nil || prevSel.ID != newSel.ID) {
		m.selectedID = newSel.ID
		cmds = append(cmds, m.loadLogPreviewCmd(newSel))
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View 渲染任务列表。
func (m Model) View() string {
	// 确保footer是最新的
	defaultHints := common.JoinKeyHelps(
		common.FormatKeyHelp(m.keys.Refresh),
		common.FormatKeyHelp(m.keys.Stop),
		common.FormatKeyHelp(m.keys.Kill),
	)
	if m.statusMsg != "" {
		m.panel.SetFooter(common.JoinKeyHelps(m.statusMsg, defaultHints))
	} else {
		m.panel.SetFooter(defaultHints)
	}

	listView := m.list.View()
	detailsView := m.renderDetails()

	content := lipgloss.JoinHorizontal(lipgloss.Top, listView, "  ", detailsView)

	// 使用Panel渲染内容
	return m.panel.Render(content)
}

func (m Model) renderDetails() string {
	task := m.selectedTask()
	if task == nil {
		return ""
	}

	// Ensure details width
	width := m.width - m.list.Width() - 4 // minus padding/gap
	if width <= 0 {
		return "" // Hide if no space
	}

	// Styles
	titleStyle := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Width(width)
	labelStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted).Width(12)
	valueStyle := lipgloss.NewStyle().Foreground(m.theme.Foreground)

	// Header
	header := titleStyle.Render(fmt.Sprintf("%s #%d", task.Command, task.ID))

	// Meta Info
	var metaRows []string

	fields := []struct{ k, v string }{
		{"Status", strings.ToUpper(task.Status)},
		{"PID", fmt.Sprintf("%v", safeDerefInt(task.PID))},
		{"WorkDir", task.WorkingDir},
		{"Started", safeFormatTime(task.StartedAt)},
		{"LogPath", task.LogFilePath},
	}

	for _, f := range fields {
		row := lipgloss.JoinHorizontal(lipgloss.Left,
			labelStyle.Render(f.k+":"),
			valueStyle.Render(f.v),
		)
		metaRows = append(metaRows, row)
	}

	metaBlock := strings.Join(metaRows, "\n")

	// Log Preview
	previewHeader := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(m.theme.Border).
		Width(width).
		Render("Recent Logs")

	preview := m.viewport.View()

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		metaBlock,
		"",
		previewHeader,
		preview,
	)
}

func safeDerefInt(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

func safeFormatTime(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.Format("15:04:05")
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

func (m Model) loadLogPreviewCmd(task *model.Task) tea.Cmd {
	if task == nil || task.LogFilePath == "" {
		return func() tea.Msg { return logPreviewMsg{taskID: 0, content: "No log file"} }
	}

	taskID := task.ID
	path := task.LogFilePath

	return func() tea.Msg {
		// Read last 1KB or ~20 lines
		content := readTail(path, 2048) // 2KB buffer
		return logPreviewMsg{
			taskID:  taskID,
			content: content,
		}
	}
}

func readTail(path string, size int64) string {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Sprintf("Error reading log: %v", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return ""
	}

	fileSize := stat.Size()
	if fileSize == 0 {
		return "(Empty log)"
	}

	start := fileSize - size
	if start < 0 {
		start = 0
	}

	if _, err := f.Seek(start, 0); err != nil {
		return ""
	}

	buf := make([]byte, size)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return ""
	}

	return string(buf[:n])
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
			key.WithKeys("x"),
			key.WithHelp("x", "停止任务"),
		),
		Kill: key.NewBinding(
			key.WithKeys("k"),
			key.WithHelp("k", "强制终止"),
		),
	}
}
