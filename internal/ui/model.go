package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aliancn/logcmd/internal/history"
	"github.com/aliancn/logcmd/internal/registry"
	"github.com/aliancn/logcmd/internal/tasks"
	"github.com/aliancn/logcmd/internal/ui/common"
	"github.com/aliancn/logcmd/internal/ui/modules/historylist"
	"github.com/aliancn/logcmd/internal/ui/modules/logviewer"
	"github.com/aliancn/logcmd/internal/ui/modules/projectlist"
	"github.com/aliancn/logcmd/internal/ui/modules/searchview"
	"github.com/aliancn/logcmd/internal/ui/modules/statspanel"
	"github.com/aliancn/logcmd/internal/ui/modules/taskmanager"
)

// SessionState 标识当前的界面状态。
type SessionState int

const (
	// ProjectListView 展示项目列表。
	ProjectListView SessionState = iota
	// HistoryListView 展示选中项目的历史记录。
	HistoryListView
	// LogViewerView 展示某条记录的日志。
	LogViewerView
	// TaskListView 展示后台任务。
	TaskListView
	// SearchView 展示全局搜索界面。
	SearchView
)

// Model 是 TUI 根 Model。
type Model struct {
	state      SessionState
	prevState  SessionState
	width      int
	height     int
	err        error
	ready      bool
	globalKeys common.GlobalKeyMap
	styles     common.Styles

	registry   *registry.Registry
	historyMgr *history.Manager
	taskMgr    *tasks.Manager

	projectList projectlist.Model
	historyList historylist.Model
	logViewer   logviewer.Model
	taskList    taskmanager.Model
	statsPanel  statspanel.Model
	searchView  searchview.Model

	projectSplitVertical bool
	projectStatsCompact  bool
}

// NewRootModel 创建根 Model。
func NewRootModel(reg *registry.Registry, historyMgr *history.Manager, taskMgr *tasks.Manager) *Model {
	styles := common.DefaultStyles()
	return &Model{
		state:       ProjectListView,
		globalKeys:  common.NewGlobalKeyMap(),
		styles:      styles,
		registry:    reg,
		historyMgr:  historyMgr,
		taskMgr:     taskMgr,
		projectList: projectlist.New(reg),
		historyList: historylist.New(historyMgr),
		logViewer:   logviewer.New(),
		taskList:    taskmanager.New(taskMgr),
		statsPanel:  statspanel.New(historyMgr),
		searchView:  searchview.New(reg),
	}
}

// Init 初始化应用。
func (m *Model) Init() tea.Cmd {
	return m.projectList.Init()
}
