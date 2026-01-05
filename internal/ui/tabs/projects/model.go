package projects

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aliancn/logcmd/internal/history"
	"github.com/aliancn/logcmd/internal/registry"
	"github.com/aliancn/logcmd/internal/ui/common"
	"github.com/aliancn/logcmd/internal/ui/components/layout"
	"github.com/aliancn/logcmd/internal/ui/modules/historylist"
	"github.com/aliancn/logcmd/internal/ui/modules/logviewer"
	"github.com/aliancn/logcmd/internal/ui/modules/projectlist"
	"github.com/aliancn/logcmd/internal/ui/modules/statspanel"
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
	statsPanel  statspanel.Model
	splitView   *layout.SplitView

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
	sPanel := statspanel.New(historyMgr, theme, styles)

	// 配置分割视图
	splitConfig := layout.DefaultSplitConfig()
	splitConfig.Ratio = 0.65 // 列表占 65%
	splitConfig.MinWindowWidth = 90 // 窄屏自动隐藏统计面板

	// 注意：这里的 wrapper 逻辑将在 Update 中处理，初始化时直接传入
	// 由于 projectlist 和 statspanel 现在 (将要) 实现 tea.Model 并返回 (tea.Model, tea.Cmd)
	// 但 Resizable 需要 SetSize，这也是满足的。
	// 唯一的问题是 Go 的接口实现检查。
	// 如果 projectList 的 Update 返回 concrete type，它不满足 tea.Model 接口 (Update签名不匹配)。
	// 我已经修改了 Update 签名，所以应该没问题。

	return Model{
		state:       ProjectListView,
		projectList: pList,
		historyList: historylist.New(historyMgr, theme, styles),
		logViewer:   logviewer.New(theme, styles),
		statsPanel:  sPanel,
		splitView:   layout.NewSplitView(&pList, &sPanel, splitConfig),
		registry:    reg,
		historyMgr:  historyMgr,
		theme:       theme,
		styles:      styles,
	}
}

// Init 初始化
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.projectList.Init(), m.splitView.Init())
}

// SetSize 设置尺寸
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	
	// 更新分割视图尺寸
	m.splitView.SetSize(width, height)
	
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
