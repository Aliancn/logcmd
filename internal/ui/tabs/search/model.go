package search

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aliancn/logcmd/internal/registry"
	"github.com/aliancn/logcmd/internal/ui/common"
	"github.com/aliancn/logcmd/internal/ui/modules/searchview"
)

// Model Search Tab的Model
type Model struct {
	searchView searchview.Model

	width  int
	height int

	registry *registry.Registry
	theme    common.Theme
	styles   common.Styles
}

// New 创建Search Tab Model
func New(reg *registry.Registry, theme common.Theme, styles common.Styles) Model {
	return Model{
		searchView: searchview.New(reg, theme, styles),
		registry:   reg,
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
	m.searchView.SetSize(width, height)
}

// GetBreadcrumbs 获取当前面包屑路径
func (m Model) GetBreadcrumbs() []string {
	return []string{"Home", "搜索"}
}
