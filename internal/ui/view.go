package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// View 渲染当前界面（新架构 - 三段式布局）
func (m *Model) View() string {
	if !m.ready {
		return "TUI 初始化中..."
	}

	// 1. 渲染Header (Removed)
	// headerView := m.header.View()

	// 2. 渲染TabBar
	tabBarView := m.tabBar.View()

	// 3. 渲染当前激活Tab的Main区域
	var mainView string
	switch m.activeTabIndex {
	case 0:
		mainView = m.projectsTab.View()
	case 1:
		mainView = m.tasksTab.View()
	case 2:
		mainView = m.searchTab.View()
	case 3:
		mainView = m.analyticsTab.View()
	default:
		mainView = m.styles.Error.Render("未知Tab")
	}

	// 4. 渲染Footer
	footerView := m.footer.View()

	// 5. 组合三段式布局
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		// headerView,
		tabBarView,
		mainView,
		footerView,
	)

	// 6. 如果有错误信息，叠加显示
	if m.err != nil {
		errView := m.styles.Error.Render(fmt.Sprintf("错误: %v", m.err))
		content = lipgloss.JoinVertical(lipgloss.Left, content, errView)
	}

	// 7. Command Palette叠加渲染（阶段4实现）
	if m.showCmdPalette {
		paletteView := m.cmdPalette.View()
		// 使用lipgloss.Place居中叠加Command Palette
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			paletteView,
			lipgloss.WithWhitespaceChars("░"),
			lipgloss.WithWhitespaceForeground(m.theme.StatusBar),
		)
	}

	return m.styles.AppContainer.Render(content)
}
