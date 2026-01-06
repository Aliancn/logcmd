package tasks

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aliancn/logcmd/internal/tasks"
	"github.com/aliancn/logcmd/internal/ui/common"
	"github.com/aliancn/logcmd/internal/ui/modules/taskmanager"
)

// Model Tasks Tab的Model
type Model struct {
	taskList taskmanager.Model

	width  int
	height int

	taskMgr *tasks.Manager
	theme   common.Theme
	styles  common.Styles
}

// New 创建Tasks Tab Model
func New(taskMgr *tasks.Manager, theme common.Theme, styles common.Styles) Model {
	return Model{
		taskList: taskmanager.New(taskMgr, theme, styles),
		taskMgr:  taskMgr,
		theme:    theme,
		styles:   styles,
	}
}

// Init 初始化
func (m Model) Init() tea.Cmd {
	return nil
}

// SetSize 设置尺寸
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.taskList.SetSize(width, height)
}

// GetBreadcrumbs 获取当前面包屑路径
func (m Model) GetBreadcrumbs() []string {
	return []string{"Home", "任务"}
}

// OnActivated 激活Tasks Tab时启动刷新
func (m *Model) OnActivated() tea.Cmd {
	return m.taskList.SetActive(true)
}

// OnDeactivated 离开Tasks Tab时停止刷新
func (m *Model) OnDeactivated() tea.Cmd {
	return m.taskList.SetActive(false)
}
