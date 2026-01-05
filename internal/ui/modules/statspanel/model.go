package statspanel

import (
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

	// UI 组件
	panel *panel.Panel

	// 布局
	width  int
	height int

	// 依赖
	theme  common.Theme
	styles common.Styles
}

// CommandStat 命令统计项
type CommandStat struct {
	Command string
	Count   int
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

	// 设置 Panel 尺寸
	m.panel.SetSize(width, height)
}

// SetProject 设置当前项目
func (m *Model) SetProject(project *model.Project) {
	m.project = project
	// 清空之前的统计数据
	m.commandDist = make(map[string]int)
	m.topCommands = make([]CommandStat, 0)
}
