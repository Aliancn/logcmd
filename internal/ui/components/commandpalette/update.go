package commandpalette

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aliancn/logcmd/internal/ui/common"
)

// Update 更新CommandPalette
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			// Esc 关闭Command Palette
			return m, func() tea.Msg {
				return common.HideCommandPaletteMsg{}
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			// Enter 执行选中的命令
			if cmd := m.GetSelectedCommand(); cmd != nil {
				// 先关闭面板，然后执行命令
				return m, tea.Batch(
					func() tea.Msg { return common.HideCommandPaletteMsg{} },
					func() tea.Msg { return cmd.Action },
				)
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "up"))):
			// 上下键导航列表
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd

		default:
			// 其他按键更新输入框
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)

			// 根据输入过滤命令
			m.filterCommands(m.input.Value())
		}
	}

	return m, tea.Batch(cmds...)
}
