package projectlist

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aliancn/logcmd/internal/model"
	"github.com/aliancn/logcmd/internal/registry"
	"github.com/aliancn/logcmd/internal/ui/common"
	"github.com/aliancn/logcmd/internal/ui/components/panel"
)

// Model 项目列表模块
type Model struct {
	// 数据层
	registry *registry.Registry
	projects []*model.Project

	// UI 组件
	list      list.Model
	panel     *panel.Panel
	pathInput textinput.Model
	nameInput textinput.Model

	// 状态
	selectedIndex    int
	isAdding         bool // 是否处于添加模式
	addStep          int  // 0: path, 1: name
	confirmingDelete bool
	deleteTargetID   int
	statusMsg        string

	// 布局
	width  int
	height int

	// 依赖
	theme  common.Theme
	styles common.Styles
	keys   keyMap
}

// keyMap 键盘绑定
type keyMap struct {
	Select  key.Binding
	Add     key.Binding
	Delete  key.Binding
	Refresh key.Binding
	Confirm key.Binding
	Cancel  key.Binding
}

// newKeyMap 创建键盘绑定
func newKeyMap() keyMap {
	return keyMap{
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "选择"),
		),
		Add: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "添加"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "删除"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "刷新"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("y", "Y", "enter"), // Adding enter for form confirm
			key.WithHelp("y/enter", "确认"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("n", "N", "esc"),
			key.WithHelp("n/esc", "取消"),
		),
	}
}

// New 创建项目列表模块
func New(reg *registry.Registry, theme common.Theme, styles common.Styles) Model {
	// 创建 list
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true

	l := list.New(nil, delegate, 0, 0)
	l.Title = "项目列表"
	l.Styles.Title = styles.Title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)

	keys := newKeyMap()
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keys.Select, keys.Add, keys.Delete}
	}

	// 创建 Inputs
	tiPath := textinput.New()
	tiPath.Placeholder = "/path/to/project"
	tiPath.Focus()
	tiPath.Width = 40

	tiName := textinput.New()
	tiName.Placeholder = "Project Name"
	tiName.Width = 30

	// 创建 Panel 布局容器
	p := panel.NewDefault("", theme, styles)

	return Model{
		registry:  reg,
		list:      l,
		panel:     p,
		pathInput: tiPath,
		nameInput: tiName,
		keys:      keys,
		theme:     theme,
		styles:    styles,
	}
}

// Init 实现 tea.Model
func (m Model) Init() tea.Cmd {
	return m.LoadProjectsCmd()
}

// SetSize 调整组件大小
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height

	// 构建 footer
	var footer string
	if m.isAdding {
		footer = m.styles.StatusBar.Render("Tab 切换输入 · Enter 下一步/确认 · Esc 取消")
	} else if m.confirmingDelete {
		// 红色边框的确认提示
		footer = m.styles.Error.Render(m.statusMsg)
	} else if m.statusMsg != "" {
		footer = m.styles.StatusBar.Render(m.statusMsg)
	} else {
		footer = m.styles.StatusBar.Render("↑/↓ 导航 · Enter 选择 · a 添加 · d 删除 · r 刷新")
	}
	m.panel.SetFooter(footer)

	// 设置 Panel 尺寸
	m.panel.SetSize(width, height)

	// 获取内容区域尺寸
	contentW, contentH := m.panel.GetContentSize()

	// 确保最小尺寸
	if contentW < 30 {
		contentW = 30
	}
	if contentH < 5 {
		contentH = 5
	}

	// 设置 list 使用精确的内容尺寸
	m.list.SetSize(contentW, contentH)
}

// CurrentProjectID 返回当前选中的项目 ID
func (m Model) CurrentProjectID() int {
	if item, ok := m.list.SelectedItem().(projectItem); ok {
		return item.project.ID
	}
	return 0
}

// CurrentProject 返回当前选中的项目
func (m Model) CurrentProject() *model.Project {
	if item, ok := m.list.SelectedItem().(projectItem); ok {
		return item.project
	}
	return nil
}
