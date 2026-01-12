package modes

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aliancn/logcmd/internal/registry"
	"github.com/aliancn/logcmd/internal/ui/common"
	"github.com/aliancn/logcmd/internal/ui/modules/projectlist"
)

// ProjectMode 项目管理模式
//
// 核心功能：
//   - 显示所有注册的项目列表
//   - 添加、删除项目
//   - 选择项目并切换到搜索模式
//   - 查看项目统计
type ProjectMode struct {
	// 依赖
	registry *registry.Registry

	// UI 组件
	projectList projectlist.Model

	// 布局
	width  int
	height int

	// 样式
	theme  common.Theme
	styles common.Styles
}

// NewProjectMode 创建项目模式
func NewProjectMode(reg *registry.Registry, theme common.Theme, styles common.Styles) *ProjectMode {
	return &ProjectMode{
		registry:    reg,
		projectList: projectlist.New(reg, theme, styles),
		theme:       theme,
		styles:      styles,
	}
}

// Name 实现 Mode 接口
func (m *ProjectMode) Name() string {
	return "project"
}

// Activate 实现 Mode 接口
func (m *ProjectMode) Activate() tea.Cmd {
	// 激活时加载项目列表
	return m.projectList.Init()
}

// Deactivate 实现 Mode 接口
func (m *ProjectMode) Deactivate() tea.Cmd {
	return nil
}

// Update 实现 Mode 接口
func (m *ProjectMode) Update(msg tea.Msg) (Mode, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.projectList.SetSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		// 检查是否在输入模式，如果是则不处理模式级快捷键
		if !m.projectList.CanUseGlobalShortcuts() {
			// 在添加/删除确认等模式下，交给projectlist处理
			var model tea.Model
			model, cmd = m.projectList.Update(msg)
			m.projectList = model.(projectlist.Model)
			return m, cmd
		}

		// 处理项目模式的特殊快捷键
		switch msg.String() {
		case "enter":
			// 选择项目并切换到搜索模式
			if proj := m.projectList.CurrentProject(); proj != nil {
				return m, func() tea.Msg {
					return SwitchModeMsg{
						ModeName: "search",
						Data:     proj, // 传递选中的项目
					}
				}
			}
		case "v":
			// 查看项目统计 - 切换到统计模式
			if proj := m.projectList.CurrentProject(); proj != nil {
				return m, func() tea.Msg {
					return SwitchModeMsg{
						ModeName: "stats",
						Data:     proj,
					}
				}
			}
		}
	}

	// 转发消息到 projectlist 组件
	var model tea.Model
	model, cmd = m.projectList.Update(msg)
	m.projectList = model.(projectlist.Model)
	return m, cmd
}

// View 实现 Mode 接口
func (m *ProjectMode) View() string {
	if m.width == 0 {
		return "初始化中..."
	}

	// 状态栏
	statusBar := m.renderStatusBar()

	// 项目列表（使用panel渲染）
	listView := m.projectList.View()

	return lipgloss.JoinVertical(lipgloss.Left,
		statusBar,
		listView,
	)
}

// HandleKey 实现 Mode 接口
func (m *ProjectMode) HandleKey(key string) (bool, tea.Cmd) {
	// 项目模式暂时没有额外的全局快捷键需要处理
	// 所有快捷键都在Update中处理
	return false, nil
}

// renderStatusBar 渲染状态栏
func (m *ProjectMode) renderStatusBar() string {
	// 获取项目数量
	projects, err := m.registry.List()
	projectCount := 0
	if err == nil {
		projectCount = len(projects)
	}

	status := fmt.Sprintf("[项目] %d 个已注册", projectCount)

	statusStyle := lipgloss.NewStyle().
		Foreground(m.theme.Foreground).
		Background(m.theme.StatusBar).
		Padding(0, 1).
		Width(m.width)

	return statusStyle.Render(status)
}
