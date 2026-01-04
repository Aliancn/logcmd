package projectlist

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aliancn/logcmd/internal/model"
	"github.com/aliancn/logcmd/internal/registry"
	"github.com/aliancn/logcmd/internal/ui/common"
)

// Model 管理项目列表视图。
type Model struct {
	list          list.Model
	registry      *registry.Registry
	keys          keyMap
	styles        common.Styles
	width         int
	height        int
	confirm       deleteConfirmState
	highlightedID int
}

// ProjectSelectedMsg 在用户选择项目时发出。
type ProjectSelectedMsg struct {
	Project *model.Project
}

// ProjectsLoadedMsg 表示项目加载完成。
type ProjectsLoadedMsg struct {
	Projects []*model.Project
	Status   string
}

// ProjectDeletedMsg 表示项目删除完成。
type ProjectDeletedMsg struct {
	ProjectID   int
	ProjectName string
	Projects    []*model.Project
}

// ProjectHighlightedMsg 表示当前选中项发生变化。
type ProjectHighlightedMsg struct {
	Project *model.Project
}

// New 创建项目列表 Model。
func New(reg *registry.Registry) Model {
	styles := common.DefaultStyles()
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "项目列表"
	l.Styles.Title = styles.Title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(true)

	k := newKeyMap()
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{k.Open, k.Refresh, k.Delete, k.Remove}
	}

	return Model{
		registry: reg,
		list:     l,
		keys:     k,
		styles:   styles,
	}
}

// Init 实现 tea.Model。
func (m Model) Init() tea.Cmd {
	return m.loadProjectsCmd()
}

// SetSize 更新列表尺寸。
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	w := width - 4
	h := height - 4
	if w < 20 {
		w = width
	}
	if h < 5 {
		h = height
	}
	m.list.SetSize(w, h)
}

// Update 处理消息。
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
	case tea.KeyMsg:
		if m.confirm.active {
			if handled, cmd := m.handleConfirmKey(msg); handled {
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
				return m, tea.Batch(cmds...)
			}
		}
		switch {
		case key.Matches(msg, m.keys.Open):
			if item, ok := m.list.SelectedItem().(projectItem); ok {
				selected := item.project
				cmds = append(cmds, func() tea.Msg {
					return ProjectSelectedMsg{Project: selected}
				})
			}
		case key.Matches(msg, m.keys.Refresh):
			cmds = append(cmds, m.loadProjectsCmd())
		case key.Matches(msg, m.keys.Delete):
			cmds = append(cmds, m.cleanProjectsCmd())
		case key.Matches(msg, m.keys.Remove):
			if project := m.selectedProject(); project != nil {
				m.confirm = deleteConfirmState{active: true, project: project}
			}
		}
	case ProjectsLoadedMsg:
		items := make([]list.Item, len(msg.Projects))
		for i, project := range msg.Projects {
			items[i] = projectItem{project: project}
		}
		m.highlightedID = 0
		m.list.SetItems(items)
		if msg.Status != "" {
			m.list.NewStatusMessage(msg.Status)
		}
	case common.ErrorMsg:
		// 透传错误
		return m, tea.Batch(cmds...)
	case ProjectDeletedMsg:
		items := make([]list.Item, len(msg.Projects))
		for i, project := range msg.Projects {
			items[i] = projectItem{project: project}
		}
		m.highlightedID = 0
		m.list.SetItems(items)
		name := msg.ProjectName
		if strings.TrimSpace(name) == "" {
			name = fmt.Sprintf("项目 #%d", msg.ProjectID)
		}
		m.list.NewStatusMessage(fmt.Sprintf("已删除 %s", name))
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)
	m.enqueueHighlightChange(&cmds)

	return m, tea.Batch(cmds...)
}

// View 渲染列表。
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "加载中..."
	}
	view := m.styles.Frame.Render(m.list.View())
	if m.confirm.active && m.confirm.project != nil {
		view = lipgloss.JoinVertical(lipgloss.Left, view, m.renderConfirmPrompt())
	}
	return view
}

func (m Model) loadProjectsCmd() tea.Cmd {
	if m.registry == nil {
		return nil
	}
	return func() tea.Msg {
		projects, err := m.registry.List()
		if err != nil {
			return common.ErrorMsg{Err: fmt.Errorf("加载项目失败: %w", err)}
		}
		return ProjectsLoadedMsg{Projects: projects}
	}
}

func (m Model) cleanProjectsCmd() tea.Cmd {
	if m.registry == nil {
		return nil
	}
	return func() tea.Msg {
		if err := m.registry.CheckAndCleanup(); err != nil {
			return common.ErrorMsg{Err: fmt.Errorf("清理项目失败: %w", err)}
		}
		projects, err := m.registry.List()
		if err != nil {
			return common.ErrorMsg{Err: fmt.Errorf("加载项目失败: %w", err)}
		}
		return ProjectsLoadedMsg{
			Projects: projects,
			Status:   "已清理无效项目",
		}
	}
}

func (m Model) deleteProjectCmd(project *model.Project) tea.Cmd {
	if m.registry == nil || project == nil {
		return nil
	}
	id := project.ID
	return func() tea.Msg {
		if err := m.registry.Delete(fmt.Sprintf("%d", id)); err != nil {
			return common.ErrorMsg{Err: fmt.Errorf("删除项目失败: %w", err)}
		}
		projects, err := m.registry.List()
		if err != nil {
			return common.ErrorMsg{Err: fmt.Errorf("加载项目失败: %w", err)}
		}
		return ProjectDeletedMsg{
			ProjectID:   id,
			ProjectName: safeProjectName(project),
			Projects:    projects,
		}
	}
}

type projectItem struct {
	project *model.Project
}

func (i projectItem) Title() string {
	name := i.project.Name
	if strings.TrimSpace(name) == "" {
		name = i.project.Path
	}
	success := fmt.Sprintf("%.1f%%", i.project.GetSuccessRate())
	return fmt.Sprintf("[%d] %s  (%s)", i.project.ID, name, success)
}

func (i projectItem) Description() string {
	last := "-"
	if i.project.LastCommandTime.Valid {
		last = i.project.LastCommandTime.Time.Format("2006-01-02 15:04:05")
	}
	return fmt.Sprintf("%s\n最后执行: %s | 总命令: %d", i.project.Path, last, i.project.TotalCommands)
}

func (i projectItem) FilterValue() string {
	return strings.Join([]string{
		strconv.Itoa(i.project.ID),
		i.project.Path,
		i.project.Name,
		strings.Join(i.project.Tags, " "),
	}, " ")
}

type keyMap struct {
	Open    key.Binding
	Refresh key.Binding
	Delete  key.Binding
	Remove  key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Open: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "查看历史"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "刷新"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "清理无效项目"),
		),
		Remove: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "删除当前项目"),
		),
	}
}

type deleteConfirmState struct {
	active  bool
	project *model.Project
}

func (m *Model) selectedProject() *model.Project {
	if item, ok := m.list.SelectedItem().(projectItem); ok {
		return item.project
	}
	return nil
}

func (m *Model) handleConfirmKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.Type {
	case tea.KeyRunes:
		for _, r := range msg.Runes {
			switch r {
			case 'y', 'Y':
				cmd := m.deleteProjectCmd(m.confirm.project)
				m.confirm = deleteConfirmState{}
				return true, cmd
			case 'n', 'N':
				m.confirm = deleteConfirmState{}
				return true, nil
			}
		}
	case tea.KeyEnter:
		cmd := m.deleteProjectCmd(m.confirm.project)
		m.confirm = deleteConfirmState{}
		return true, cmd
	case tea.KeyEsc:
		m.confirm = deleteConfirmState{}
		return true, nil
	}
	return true, nil
}

func (m *Model) enqueueHighlightChange(cmds *[]tea.Cmd) {
	current := m.selectedProject()
	currentID := 0
	if current != nil {
		currentID = current.ID
	}
	if currentID == m.highlightedID {
		return
	}
	m.highlightedID = currentID
	project := current
	*cmds = append(*cmds, func() tea.Msg {
		return ProjectHighlightedMsg{Project: project}
	})
}

func (m Model) renderConfirmPrompt() string {
	if m.confirm.project == nil {
		return ""
	}
	p := m.confirm.project
	info := fmt.Sprintf("确定删除项目 #%d?\n路径: %s\n[y] 确认  [n] 取消", p.ID, p.Path)
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("204")).
		Padding(1, 2).
		Width(max(40, m.width/2)).
		Render(info)
	return box
}

func safeProjectName(project *model.Project) string {
	if project == nil {
		return ""
	}
	name := strings.TrimSpace(project.Name)
	if name != "" {
		return name
	}
	return project.Path
}

// SelectedProject 暴露当前选中的项目。
func (m Model) SelectedProject() *model.Project {
	return m.selectedProject()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
