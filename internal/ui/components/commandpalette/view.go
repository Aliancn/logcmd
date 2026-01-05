package commandpalette

import (
	"github.com/charmbracelet/lipgloss"
)

// View 渲染CommandPalette
func (m Model) View() string {
	if !m.active {
		return ""
	}

	// 输入框
	inputView := m.input.View()

	// 命令列表
	listView := m.list.View()

	// 组合输入框和列表
	content := lipgloss.JoinVertical(lipgloss.Left, inputView, "", listView)

	// 使用圆角边框包裹，添加标题
	paletteStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Primary).
		Padding(1).
		Width(m.input.Width + 4) // 加上padding

	title := m.styles.Title.Render("⚡ 命令面板")

	return paletteStyle.Render(lipgloss.JoinVertical(lipgloss.Left, title, "", content))
}
