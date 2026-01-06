package projects

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aliancn/logcmd/internal/history"
	"github.com/aliancn/logcmd/internal/registry"
	"github.com/aliancn/logcmd/internal/ui/common"
	"github.com/aliancn/logcmd/internal/ui/modules/historylist"
	"github.com/aliancn/logcmd/internal/ui/modules/logviewer"
	"github.com/aliancn/logcmd/internal/ui/modules/projectlist"
)

// ProjectsViewState 项目Tab的子状态
type ProjectsViewState int

const (
	ProjectListView ProjectsViewState = iota // 项目列表
	HistoryListView                          // 历史记录
	LogViewerView                            // 日志查看
)

// Model Projects Tab的Model
type Model struct {
	state ProjectsViewState // 当前子状态

	// 子模块
	projectList projectlist.Model
	historyList historylist.Model
	logViewer   logviewer.Model

	// 布局状态
	width  int
	height int

	// 依赖
	registry   *registry.Registry
	historyMgr *history.Manager

	theme  common.Theme
	styles common.Styles
}

// New 创建Projects Tab Model
func New(reg *registry.Registry, historyMgr *history.Manager, theme common.Theme, styles common.Styles) Model {
	pList := projectlist.New(reg, theme, styles)

	return Model{
		state:       ProjectListView,
		projectList: pList,
		historyList: historylist.New(historyMgr, theme, styles),
		logViewer:   logviewer.New(theme, styles),
		registry:    reg,
		historyMgr:  historyMgr,
		theme:       theme,
		styles:      styles,
	}
}

// Init 初始化
func (m Model) Init() tea.Cmd {
	return m.projectList.Init()
}

// SetSize 设置尺寸
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height

	// 项目列表占满全屏
	m.projectList.SetSize(width, height)

	// 其他视图使用全尺寸
	m.historyList.SetSize(width, height)
	m.logViewer.SetSize(width, height)
}

// GetBreadcrumbs 获取当前面包屑路径
func (m Model) GetBreadcrumbs() []string {
	switch m.state {
	case ProjectListView:
		return []string{"Home", "项目"}
	case HistoryListView:
		return []string{"Home", "项目", "历史记录"}
	case LogViewerView:
		return []string{"Home", "项目", "历史记录", "日志"}
	default:
		return []string{"Home", "项目"}
	}
}

// AllowTaskShortcut 当前状态下是否允许使用Tab快捷键跳转到任务视图
func (m Model) AllowTaskShortcut() bool {
	if m.state != ProjectListView {
		return true
	}
	return m.projectList.CanUseGlobalShortcuts()
}
