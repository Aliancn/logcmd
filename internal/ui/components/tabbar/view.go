package tabbar

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// View 渲染TabBar
func (m Model) View() string {
	if m.width == 0 {
		return ""
	}

	// 渲染每个Tab标签
	var tabs []string
	for _, tab := range m.tabs {
		var tabStr string
		if tab.Index == m.activeIndex {
			// 激活的Tab：使用高亮样式
			tabStr = m.styles.TabActive.Render(fmt.Sprintf("[%s] %s", tab.Key, tab.Label))
		} else {
			// 非激活的Tab：使用普通样式
			tabStr = m.styles.TabInactive.Render(fmt.Sprintf("[%s] %s", tab.Key, tab.Label))
		}
		tabs = append(tabs, tabStr)
	}

	// 使用空格分隔Tab
	tabsLine := lipgloss.JoinHorizontal(lipgloss.Left, tabs...)

	// 在激活的Tab下方添加下划线指示器
	var underlines []string
	for _, tab := range m.tabs {
		tabWidth := lipgloss.Width(fmt.Sprintf("[%s] %s", tab.Key, tab.Label)) + 4 // +4 for padding
		if tab.Index == m.activeIndex {
			// 激活Tab：显示下划线
			underlineStyle := lipgloss.NewStyle().
				Foreground(m.theme.TabActive).
				Width(tabWidth)
			underlines = append(underlines, underlineStyle.Render("───────"))
		} else {
			// 非激活Tab：空白
			underlines = append(underlines, lipgloss.NewStyle().Width(tabWidth).Render(""))
		}
	}

	underlinesLine := lipgloss.JoinHorizontal(lipgloss.Left, underlines...)

	// 组合Tab标签和下划线
	return lipgloss.JoinVertical(lipgloss.Left, tabsLine, underlinesLine)
}
