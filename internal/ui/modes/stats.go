package modes

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aliancn/logcmd/internal/history"
	"github.com/aliancn/logcmd/internal/model"
	"github.com/aliancn/logcmd/internal/ui/common"
	"github.com/aliancn/logcmd/internal/ui/modules/statspanel"
)

// StatsMode 统计分析模式
//
// 核心功能：
//   - 显示项目执行统计
//   - 命令执行频率分析
//   - 成功/失败率统计
//   - 支持全局和单项目统计切换
type StatsMode struct {
	// 依赖
	historyMgr *history.Manager

	// UI 组件
	statsPanel statspanel.Model

	// 状态
	currentProject *model.Project // nil = 全局统计
	isGlobal       bool           // 是否显示全局统计

	// 布局
	width  int
	height int

	// 样式
	theme  common.Theme
	styles common.Styles
}

// NewStatsMode 创建统计模式
func NewStatsMode(historyMgr *history.Manager, theme common.Theme, styles common.Styles) *StatsMode {
	return &StatsMode{
		historyMgr: historyMgr,
		statsPanel: statspanel.New(historyMgr, theme, styles),
		isGlobal:   true, // 默认显示全局统计
		theme:      theme,
		styles:     styles,
	}
}

// Name 实现 Mode 接口
func (m *StatsMode) Name() string {
	return "stats"
}

// Activate 实现 Mode 接口
func (m *StatsMode) Activate() tea.Cmd {
	// 激活时刷新统计数据
	return m.statsPanel.Refresh()
}

// Deactivate 实现 Mode 接口
func (m *StatsMode) Deactivate() tea.Cmd {
	return nil
}

// Update 实现 Mode 接口
func (m *StatsMode) Update(msg tea.Msg) (Mode, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.statsPanel.SetSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		// 处理统计模式的特殊快捷键
		switch msg.String() {
		case "a":
			// 切换全局/项目统计
			m.toggleScope()
			return m, m.statsPanel.Refresh()
		case "r":
			// 手动刷新
			return m, m.statsPanel.Refresh()
		}
	}

	// 转发消息到 statsPanel 组件
	var model tea.Model
	model, cmd = m.statsPanel.Update(msg)
	m.statsPanel = model.(statspanel.Model)
	return m, cmd
}

// View 实现 Mode 接口
func (m *StatsMode) View() string {
	if m.width == 0 {
		return "初始化中..."
	}

	// 状态栏
	statusBar := m.renderStatusBar()

	// 统计面板（使用panel渲染）
	statsView := m.statsPanel.View()

	return lipgloss.JoinVertical(lipgloss.Left,
		statusBar,
		statsView,
	)
}

// HandleKey 实现 Mode 接口
func (m *StatsMode) HandleKey(key string) (bool, tea.Cmd) {
	// 统计模式的快捷键在Update中处理
	return false, nil
}

// SetProject 设置当前项目（用于限定统计范围）
func (m *StatsMode) SetProject(proj *model.Project) {
	m.currentProject = proj
	if proj != nil {
		m.isGlobal = false
		m.statsPanel.SetProject(proj)
	} else {
		m.isGlobal = true
		m.statsPanel.SetProject(nil)
	}
}

// toggleScope 切换统计范围
func (m *StatsMode) toggleScope() {
	m.isGlobal = !m.isGlobal
	if m.isGlobal {
		// 切换到全局统计
		m.statsPanel.SetProject(nil)
	} else {
		// 切换到项目统计（如果有选中的项目）
		m.statsPanel.SetProject(m.currentProject)
	}
}

// renderStatusBar 渲染状态栏
func (m *StatsMode) renderStatusBar() string {
	scope := "全局统计"
	if !m.isGlobal && m.currentProject != nil {
		scope = fmt.Sprintf("项目 #%d 统计", m.currentProject.ID)
	}

	status := fmt.Sprintf("[统计] %s", scope)

	statusStyle := lipgloss.NewStyle().
		Foreground(m.theme.Foreground).
		Background(m.theme.StatusBar).
		Padding(0, 1).
		Width(m.width)

	return statusStyle.Render(status)
}
