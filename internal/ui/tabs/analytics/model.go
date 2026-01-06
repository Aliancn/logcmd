package analytics

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aliancn/logcmd/internal/history"
	"github.com/aliancn/logcmd/internal/ui/common"
	"github.com/aliancn/logcmd/internal/ui/modules/statspanel"
)

// Model Analytics Tab的Model
type Model struct {
	statsPanel statspanel.Model

	width  int
	height int

	historyMgr *history.Manager
	theme      common.Theme
	styles     common.Styles
}

// New 创建Analytics Tab Model
func New(historyMgr *history.Manager, theme common.Theme, styles common.Styles) Model {
	return Model{
		statsPanel: statspanel.New(historyMgr, theme, styles),
		historyMgr: historyMgr,
		theme:      theme,
		styles:     styles,
	}
}

// Init 初始化
func (m Model) Init() tea.Cmd {
	return nil
}

// SetSize 设置尺寸
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.statsPanel.SetSize(width, height)
}

// GetBreadcrumbs 获取当前面包屑路径
func (m Model) GetBreadcrumbs() []string {
	return []string{"Home", "统计"}
}

// OnActivated 激活统计Tab时刷新数据
func (m *Model) OnActivated() tea.Cmd {
	return m.statsPanel.Refresh()
}

// OnDeactivated 离开统计Tab时占位
func (m *Model) OnDeactivated() tea.Cmd {
	return nil
}
