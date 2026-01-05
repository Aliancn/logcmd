package commandpalette

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aliancn/logcmd/internal/ui/common"
)

// Command 命令定义
type Command struct {
	ID     string  // 命令ID
	Label  string  // 显示标签
	Desc   string  // 描述
	Action tea.Msg // 执行的动作（消息）
}

// Title 实现list.Item接口
func (c Command) Title() string { return c.Label }

// Description 实现list.Item接口
func (c Command) Description() string { return c.Desc }

// FilterValue 实现list.Item接口（用于模糊搜索）
func (c Command) FilterValue() string { return c.Label + " " + c.Desc }

// Model CommandPalette组件的Model
type Model struct {
	input     textinput.Model // 输入框
	list      list.Model      // 命令列表
	commands  []Command       // 所有命令
	active    bool            // 是否激活显示
	width     int             // 组件宽度
	height    int             // 组件高度
	theme     common.Theme
	styles    common.Styles
}

// New 创建CommandPalette Model
func New(theme common.Theme, styles common.Styles) Model {
	// 创建输入框
	input := textinput.New()
	input.Placeholder = "输入命令..."
	input.Focus()
	input.CharLimit = 50
	input.Width = 50

	// 创建列表（先用空列表初始化）
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = ""
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)

	return Model{
		input:    input,
		list:     l,
		commands: []Command{},
		active:   false,
		theme:    theme,
		styles:   styles,
	}
}

// SetCommands 设置命令列表
func (m *Model) SetCommands(commands []Command) {
	m.commands = commands

	// 转换为list.Item
	items := make([]list.Item, len(commands))
	for i, cmd := range commands {
		items[i] = cmd
	}
	m.list.SetItems(items)
}

// SetSize 设置尺寸
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height

	// 命令面板占屏幕的2/3宽度和1/2高度
	paletteWidth := min(width*common.CommandPaletteWidthRatio/100, common.CommandPaletteMaxWidth)
	paletteHeight := min(height*common.CommandPaletteHeightRatio/100, common.CommandPaletteMaxHeight)

	// 输入框宽度为面板宽度-4（边距）
	m.input.Width = paletteWidth - 4

	// 列表高度为面板高度-6（输入框+边框+间距）
	listHeight := max(paletteHeight-6, 5)
	m.list.SetSize(paletteWidth-4, listHeight)
}

// Activate 激活Command Palette
func (m *Model) Activate() {
	m.active = true
	m.input.Focus()
	m.input.SetValue("")
	m.list.ResetFilter()
}

// Deactivate 关闭Command Palette
func (m *Model) Deactivate() {
	m.active = false
	m.input.Blur()
}

// IsActive 是否激活
func (m Model) IsActive() bool {
	return m.active
}

// GetSelectedCommand 获取当前选中的命令
func (m Model) GetSelectedCommand() *Command {
	if item := m.list.SelectedItem(); item != nil {
		if cmd, ok := item.(Command); ok {
			return &cmd
		}
	}
	return nil
}

// filterCommands 根据输入过滤命令
func (m *Model) filterCommands(query string) {
	if query == "" {
		// 无输入时显示所有命令
		items := make([]list.Item, len(m.commands))
		for i, cmd := range m.commands {
			items[i] = cmd
		}
		m.list.SetItems(items)
		return
	}

	// 简单的模糊匹配：命令标签或描述包含查询字符串
	query = strings.ToLower(query)
	var filtered []list.Item
	for _, cmd := range m.commands {
		if strings.Contains(strings.ToLower(cmd.Label), query) ||
			strings.Contains(strings.ToLower(cmd.Desc), query) {
			filtered = append(filtered, cmd)
		}
	}
	m.list.SetItems(filtered)
}
