package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aliancn/logcmd/internal/history"
	"github.com/aliancn/logcmd/internal/registry"
	"github.com/aliancn/logcmd/internal/tasks"
	"github.com/aliancn/logcmd/internal/ui/common"
	"github.com/aliancn/logcmd/internal/ui/components/commandpalette"
	"github.com/aliancn/logcmd/internal/ui/components/footer"
	"github.com/aliancn/logcmd/internal/ui/components/tabbar"
	"github.com/aliancn/logcmd/internal/ui/tabs/analytics"
	"github.com/aliancn/logcmd/internal/ui/tabs/projects"
	"github.com/aliancn/logcmd/internal/ui/tabs/search"
	tasksTab "github.com/aliancn/logcmd/internal/ui/tabs/tasks"
)

// Model 是TUI根Model（新架构）
type Model struct {
	// 三段式布局组件
	tabBar     tabbar.Model
	footer     footer.Model
	cmdPalette commandpalette.Model

	// 4个Tab容器
	projectsTab  projects.Model
	tasksTab     tasksTab.Model
	searchTab    search.Model
	analyticsTab analytics.Model

	// 全局状态
	activeTabIndex      int      // 当前激活的Tab索引（0-3）
	lastNonTaskTabIndex int      // Tab快捷键返回的目标索引
	showCmdPalette      bool     // Command Palette显示状态（阶段4实现）
	breadcrumbs         []string // 面包屑导航
	width               int
	height              int
	ready               bool
	err                 error

	// 依赖注入（保持不变）
	registry   *registry.Registry
	historyMgr *history.Manager
	taskMgr    *tasks.Manager

	// 样式和配置
	theme      common.Theme
	styles     common.Styles
	globalKeys common.GlobalKeyMap
}

// NewRootModel 创建根Model（新架构）
func NewRootModel(reg *registry.Registry, historyMgr *history.Manager, taskMgr *tasks.Manager) *Model {
	theme := common.DefaultTheme()
	styles := common.NewStyles(theme)
	globalKeys := common.NewGlobalKeyMap()
	footerModel := footer.New(theme, styles)
	footerModel.SetHints(common.GlobalFooterHints(globalKeys))

	// 创建Command Palette并设置命令
	cmdPalette := commandpalette.New(theme, styles)
	cmdPalette.SetCommands(commandpalette.DefaultCommands())

	return &Model{
		// 初始化全局组件
		tabBar:     tabbar.New(theme, styles),
		footer:     footerModel,
		cmdPalette: cmdPalette,

		// 初始化4个Tab容器
		projectsTab:  projects.New(reg, historyMgr, theme, styles),
		tasksTab:     tasksTab.New(taskMgr, theme, styles),
		searchTab:    search.New(reg, theme, styles),
		analyticsTab: analytics.New(historyMgr, theme, styles),

		// 全局状态
		activeTabIndex:      0, // 默认激活第一个Tab（项目列表）
		lastNonTaskTabIndex: 0,
		showCmdPalette:      false,
		breadcrumbs:         []string{"Home", "Projects"},

		// 依赖注入
		registry:   reg,
		historyMgr: historyMgr,
		taskMgr:    taskMgr,

		// 样式和配置
		theme:      theme,
		styles:     styles,
		globalKeys: globalKeys,
	}
}

// Init 初始化应用
func (m *Model) Init() tea.Cmd {
	// 初始化第一个Tab（项目列表）
	return m.projectsTab.Init()
}

// calculateMainAreaSize 计算Main区域的尺寸
func (m *Model) calculateMainAreaSize() (width, height int) {
	width = m.width
	height = max(m.height-common.TotalOverhead, common.MinMainAreaHeight)
	return
}

// updateBreadcrumbs 根据当前Tab更新面包屑
func (m *Model) updateBreadcrumbs() {
	switch m.activeTabIndex {
	case 0:
		m.breadcrumbs = m.projectsTab.GetBreadcrumbs()
	case 1:
		m.breadcrumbs = m.tasksTab.GetBreadcrumbs()
	case 2:
		m.breadcrumbs = m.searchTab.GetBreadcrumbs()
	case 3:
		m.breadcrumbs = m.analyticsTab.GetBreadcrumbs()
	default:
		m.breadcrumbs = []string{"Home"}
	}
}

func (m *Model) canTriggerTaskShortcut() bool {
	if m.showCmdPalette {
		return false
	}
	if m.activeTabIndex == 0 { // Projects Tab is now index 0
		return m.projectsTab.AllowTaskShortcut()
	}
	return true
}

func (m *Model) taskShortcutTarget() int {
	if m.activeTabIndex == 1 { // Tasks Tab is now index 1
		target := m.lastNonTaskTabIndex
		if target == 1 || target < 0 || target > 3 {
			return 0 // Default back to Projects
		}
		return target
	}
	if m.activeTabIndex >= 0 && m.activeTabIndex < 4 {
		m.lastNonTaskTabIndex = m.activeTabIndex
	}
	return 1 // Go to Tasks
}

func (m *Model) handleTaskBack(cmds *[]tea.Cmd) bool {
	if m.activeTabIndex != 1 || m.showCmdPalette { // Tasks Tab is index 1
		return false
	}
	target := m.taskShortcutTarget()
	if target == 1 {
		target = 0 // Default back to Projects
	}
	*cmds = append(*cmds, func() tea.Msg {
		return common.SwitchTabMsg{Index: target}
	})
	return true
}

func (m *Model) handleTabDeactivated(index int) tea.Cmd {
	switch index {
	case 1:
		return m.tasksTab.OnDeactivated()
	case 2:
		return m.searchTab.OnDeactivated()
	case 3:
		return m.analyticsTab.OnDeactivated()
	}
	return nil
}

func (m *Model) handleTabActivated(index int) tea.Cmd {
	switch index {
	case 1:
		return m.tasksTab.OnActivated()
	case 2:
		return m.searchTab.OnActivated()
	case 3:
		return m.analyticsTab.OnActivated()
	}
	return nil
}
