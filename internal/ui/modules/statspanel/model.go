package statspanel

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/aliancn/logcmd/internal/history"
	"github.com/aliancn/logcmd/internal/model"
	"github.com/aliancn/logcmd/internal/ui/common"
	"github.com/aliancn/logcmd/internal/ui/components/panel"
)

// Model 统计面板模块
type Model struct {
	// 数据层
	historyMgr *history.Manager
	project    *model.Project

	// 统计数据缓存
	commandDist map[string]int
	topCommands []CommandStat
	summary     statsSummary
	failures    []recentFailure
	lastUpdated time.Time

	// UI 组件
	panel *panel.Panel

	// 布局
	width  int
	height int

	// 依赖
	theme  common.Theme
	styles common.Styles

	loading bool
	err     error
}

// CommandStat 命令统计项
type CommandStat struct {
	Command string
	Count   int
}

type statsSummary struct {
	Total       int
	Success     int
	Failed      int
	AvgDuration time.Duration
	LastRun     time.Time
}

type recentFailure struct {
	ProjectID int
	Command   string
	Status    string
	ExitCode  int
	StartedAt time.Time
	Duration  time.Duration
}

// New 创建统计面板
func New(historyMgr *history.Manager, theme common.Theme, styles common.Styles) Model {
	// 创建 Panel 布局容器（紫色边框）
	p := panel.NewDefault("", theme, styles)

	return Model{
		historyMgr:  historyMgr,
		panel:       p,
		theme:       theme,
		styles:      styles,
		commandDist: make(map[string]int),
		topCommands: make([]CommandStat, 0),
		failures:    make([]recentFailure, 0),
	}
}

// Init 实现 tea.Model
func (m Model) Init() tea.Cmd {
	return nil
}

// SetSize 设置尺寸
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.panel.SetFooter("当前视图暂无快捷键")

	// 设置 Panel 尺寸
	m.panel.SetSize(width, height)
}

// SetProject 设置当前项目
func (m *Model) SetProject(project *model.Project) {
	m.project = project
	// 清空之前的统计数据
	m.commandDist = make(map[string]int)
	m.topCommands = make([]CommandStat, 0)
	m.summary = statsSummary{}
	m.failures = make([]recentFailure, 0)
	m.lastUpdated = time.Time{}
	m.err = nil
}

// Refresh 主动刷新统计数据
func (m *Model) Refresh() tea.Cmd {
	m.loading = true
	m.err = nil
	return m.LoadStatsCmd()
}

func (m Model) currentProjectID() int {
	if m.project == nil {
		return 0
	}
	return m.project.ID
}
