package modes

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aliancn/logcmd/internal/tasks"
	"github.com/aliancn/logcmd/internal/ui/common"
	"github.com/aliancn/logcmd/internal/ui/modules/taskmanager"
)

// TaskMode 任务管理模式
//
// 核心功能：
//   - 显示运行中的后台任务
//   - 实时查看任务日志
//   - 停止或终止任务
//   - 自动刷新任务状态
type TaskMode struct {
	// 依赖
	taskMgr *tasks.Manager

	// UI 组件
	taskManager taskmanager.Model

	// 布局
	width  int
	height int

	// 样式
	theme  common.Theme
	styles common.Styles
}

// NewTaskMode 创建任务模式
func NewTaskMode(taskMgr *tasks.Manager, theme common.Theme, styles common.Styles) *TaskMode {
	return &TaskMode{
		taskMgr:     taskMgr,
		taskManager: taskmanager.New(taskMgr, theme, styles),
		theme:       theme,
		styles:      styles,
	}
}

// Name 实现 Mode 接口
func (m *TaskMode) Name() string {
	return "task"
}

// Activate 实现 Mode 接口
func (m *TaskMode) Activate() tea.Cmd {
	// 激活任务管理器，启动自动刷新
	return m.taskManager.SetActive(true)
}

// Deactivate 实现 Mode 接口
func (m *TaskMode) Deactivate() tea.Cmd {
	// 停用自动刷新
	return m.taskManager.SetActive(false)
}

// Update 实现 Mode 接口
func (m *TaskMode) Update(msg tea.Msg) (Mode, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.taskManager.SetSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		// 处理任务模式的特殊快捷键
		switch msg.String() {
		case "enter":
			// TODO: 实现日志实时查看功能
			// 当前taskmanager已经有viewport预览，这里可以扩展为全屏日志查看
			// 暂时使用taskmanager默认的预览功能
		}
	}

	// 转发消息到 taskmanager 组件
	m.taskManager, cmd = m.taskManager.Update(msg)
	return m, cmd
}

// View 实现 Mode 接口
func (m *TaskMode) View() string {
	if m.width == 0 {
		return "初始化中..."
	}

	// 状态栏
	statusBar := m.renderStatusBar()

	// 任务管理器（使用panel渲染）
	taskView := m.taskManager.View()

	return lipgloss.JoinVertical(lipgloss.Left,
		statusBar,
		taskView,
	)
}

// HandleKey 实现 Mode 接口
func (m *TaskMode) HandleKey(key string) (bool, tea.Cmd) {
	// 任务模式暂时没有额外的全局快捷键需要处理
	// x 和 k 已经在 taskmanager 内部处理
	return false, nil
}

// renderStatusBar 渲染状态栏
func (m *TaskMode) renderStatusBar() string {
	// 获取活跃任务数量
	tasks, err := m.taskMgr.ListActive()
	taskCount := 0
	if err == nil {
		taskCount = len(tasks)
	}

	status := fmt.Sprintf("[任务] %d 个运行中", taskCount)

	statusStyle := lipgloss.NewStyle().
		Foreground(m.theme.Foreground).
		Background(m.theme.StatusBar).
		Padding(0, 1).
		Width(m.width)

	return statusStyle.Render(status)
}
