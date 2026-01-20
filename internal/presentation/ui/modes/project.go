package modes

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aliancn/logcmd/internal/platform/history"
	"github.com/aliancn/logcmd/internal/domain/model"
	"github.com/aliancn/logcmd/internal/platform/registry"
	"github.com/aliancn/logcmd/internal/presentation/ui/common"
	"github.com/aliancn/logcmd/internal/presentation/ui/modules/historylist"
	"github.com/aliancn/logcmd/internal/presentation/ui/modules/projectlist"
)

// ProjectMode 项目管理模式
//
// 核心功能：
//   - 显示所有注册的项目列表
//   - 添加、删除项目
//   - 选择项目并切换到搜索模式
//   - 查看项目统计
type projectViewState int

const (
	projectViewList projectViewState = iota
	projectViewHistory
)

// ProjectReturnState 表示切回项目模式时需要恢复的状态。
type ProjectReturnState struct {
	SelectedProject *model.Project
	ShowHistory     bool
}

type ProjectMode struct {
	// 依赖
	registry *registry.Registry

	// UI 组件
	projectList projectlist.Model
	historyList historylist.Model

	// 布局
	width  int
	height int

	// 状态
	viewState       projectViewState
	selectedProject *model.Project
	pendingState    *ProjectReturnState

	// 样式
	theme  common.Theme
	styles common.Styles
}

// NewProjectMode 创建项目模式
func NewProjectMode(reg *registry.Registry, historyMgr *history.Manager, theme common.Theme, styles common.Styles) *ProjectMode {
	return &ProjectMode{
		registry:    reg,
		projectList: projectlist.New(reg, theme, styles),
		historyList: historylist.New(historyMgr, theme, styles),
		viewState:   projectViewList,
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
	if m.pendingState != nil {
		state := m.pendingState
		m.pendingState = nil
		if state.ShowHistory && state.SelectedProject != nil {
			return m.enterHistoryView(state.SelectedProject)
		}
	}

	// 激活时加载项目列表
	m.viewState = projectViewList
	m.selectedProject = nil
	m.historyList.SetProject(nil)
	return m.projectList.Init()
}

// Deactivate 实现 Mode 接口
func (m *ProjectMode) Deactivate() tea.Cmd {
	return nil
}

// Update 实现 Mode 接口
func (m *ProjectMode) Update(msg tea.Msg) (Mode, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typed.Width
		m.height = typed.Height
		m.resizeComponents()
		return m, nil

	case tea.KeyMsg:
		if m.viewState == projectViewHistory {
			if typed.Type == tea.KeyEsc || typed.String() == "esc" {
				return m, m.exitHistoryView()
			}
			if typed.String() == "v" && m.selectedProject != nil {
				return m, func() tea.Msg {
					return SwitchModeMsg{
						ModeName: "stats",
						Data:     m.selectedProject,
					}
				}
			}
		} else {
			// 检查是否在输入模式，如果是则不处理模式级快捷键
			if !m.projectList.CanUseGlobalShortcuts() {
				var model tea.Model
				var cmd tea.Cmd
				model, cmd = m.projectList.Update(msg)
				m.projectList = model.(projectlist.Model)
				return m, cmd
			}

			switch typed.String() {
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

	case projectlist.ProjectSelectedMsg:
		return m, m.enterHistoryView(typed.Project)

	case projectlist.ProjectDeletedMsg:
		if m.selectedProject != nil && typed.ProjectID == m.selectedProject.ID {
			return m, m.exitHistoryView()
		}

	case historylist.OpenLogMsg:
		if typed.History != nil && typed.History.LogFilePath != "" {
			h := typed.History
			returnState := m.historyReturnState()
			return m, func() tea.Msg {
				return OpenLogFileMsg{
					FilePath:    h.LogFilePath,
					LineNum:     0,
					SearchQuery: h.CommandName,
					ReturnMode:  "project",
					ReturnData:  returnState,
					Follow:      false,
				}
			}
		}
	}

	var cmd tea.Cmd
	switch m.viewState {
	case projectViewHistory:
		m.historyList, cmd = m.historyList.Update(msg)
	default:
		var model tea.Model
		model, cmd = m.projectList.Update(msg)
		m.projectList = model.(projectlist.Model)
	}
	return m, cmd
}

// View 实现 Mode 接口
func (m *ProjectMode) View() string {
	if m.width == 0 {
		return "初始化中..."
	}

	// 状态栏
	statusBar := m.renderStatusBar()

	var mainView string
	if m.viewState == projectViewHistory {
		mainView = m.historyList.View()
	} else {
		mainView = m.projectList.View()
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		statusBar,
		mainView,
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
	statusStyle := lipgloss.NewStyle().
		Foreground(m.theme.Foreground).
		Background(m.theme.StatusBar).
		Padding(0, 1).
		Width(m.width)

	if m.viewState == projectViewHistory {
		status := "[运行记录] 未选择项目 · esc 返回"
		if m.selectedProject != nil {
			status = fmt.Sprintf("[运行记录] #%d %s · enter 查看日志 · r 刷新 · f 筛选 · esc 返回",
				m.selectedProject.ID,
				m.selectedProject.Name)
		}
		return statusStyle.Render(status)
	}

	projectCount := m.projectCount()
	status := fmt.Sprintf("[项目] %d 个已注册 · enter 运行记录 · a 添加 · d 删除 · v 统计", projectCount)
	return statusStyle.Render(status)
}

func (m *ProjectMode) projectCount() int {
	if m.registry == nil {
		return 0
	}
	projects, err := m.registry.List()
	if err != nil {
		return 0
	}
	return len(projects)
}

func (m *ProjectMode) resizeComponents() {
	if m.width == 0 || m.height == 0 {
		return
	}
	contentHeight := m.height - 1
	if contentHeight < 1 {
		contentHeight = m.height
	}
	m.projectList.SetSize(m.width, contentHeight)
	m.historyList.SetSize(m.width, contentHeight)
}

func (m *ProjectMode) enterHistoryView(proj *model.Project) tea.Cmd {
	if proj == nil {
		return nil
	}
	m.viewState = projectViewHistory
	m.selectedProject = proj
	m.historyList.SetProject(proj)
	m.resizeComponents()
	return m.historyList.LoadHistoryCmd()
}

func (m *ProjectMode) exitHistoryView() tea.Cmd {
	m.viewState = projectViewList
	m.selectedProject = nil
	m.historyList.SetProject(nil)
	m.resizeComponents()
	return nil
}

// SetReturnState 设置待恢复的视图状态。
func (m *ProjectMode) SetReturnState(state *ProjectReturnState) {
	m.pendingState = state
}

func (m *ProjectMode) historyReturnState() *ProjectReturnState {
	if m.viewState != projectViewHistory || m.selectedProject == nil {
		return nil
	}
	projectCopy := *m.selectedProject
	return &ProjectReturnState{
		SelectedProject: &projectCopy,
		ShowHistory:     true,
	}
}
