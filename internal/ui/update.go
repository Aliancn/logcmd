package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aliancn/logcmd/internal/ui/common"
)

// Update 是根Model的Update实现（新架构）
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = typed.Width, typed.Height
		m.ready = true

		// 计算Main区域尺寸
		mainWidth, mainHeight := m.calculateMainAreaSize()

		// 分发尺寸到所有组件
		// m.header.SetSize(m.width)
		m.tabBar.SetSize(m.width)
		m.footer.SetSize(m.width)
		m.cmdPalette.SetSize(m.width, m.height)

		// 分发尺寸到所有Tab容器
		m.projectsTab.SetSize(mainWidth, mainHeight)
		m.tasksTab.SetSize(mainWidth, mainHeight)
		m.searchTab.SetSize(mainWidth, mainHeight)
		m.analyticsTab.SetSize(mainWidth, mainHeight)

	case tea.KeyMsg:
		// 处理全局快捷键
		if m.handleGlobalKey(typed, &cmds) {
			break
		}

	case common.SwitchTabMsg:
		// 处理Tab切换消息
		if typed.Index >= 0 && typed.Index < 4 {
			m.activeTabIndex = typed.Index
			m.tabBar.SetActive(typed.Index)
			m.updateBreadcrumbs()
			// m.header.SetBreadcrumbs(m.breadcrumbs)
		}

	case common.UpdateBreadcrumbsMsg:
		// 更新面包屑
		m.breadcrumbs = typed.Items
		// m.header.SetBreadcrumbs(m.breadcrumbs)

	case common.ShowCommandPaletteMsg:
		// 显示Command Palette
		m.showCmdPalette = true
		m.cmdPalette.Activate()

	case common.HideCommandPaletteMsg:
		// 隐藏Command Palette
		m.showCmdPalette = false
		m.cmdPalette.Deactivate()

	case common.ErrorMsg:
		m.err = typed.Err
	}

	// 如果Command Palette激活，优先路由消息给它
	if m.showCmdPalette {
		var cmd tea.Cmd
		m.cmdPalette, cmd = m.cmdPalette.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		// 否则路由消息到当前激活的Tab
		var cmd tea.Cmd
		switch m.activeTabIndex {
		case 0: // Projects Tab
			m.projectsTab, cmd = m.projectsTab.Update(msg)
		case 1: // Tasks Tab
			m.tasksTab, cmd = m.tasksTab.Update(msg)
		case 2: // Search Tab
			m.searchTab, cmd = m.searchTab.Update(msg)
		case 3: // Analytics Tab
			m.analyticsTab, cmd = m.analyticsTab.Update(msg)
		}
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleGlobalKey 处理全局快捷键
func (m *Model) handleGlobalKey(keyMsg tea.KeyMsg, cmds *[]tea.Cmd) bool {
	switch {
	case key.Matches(keyMsg, m.globalKeys.Quit):
		// Ctrl+C 退出
		*cmds = append(*cmds, tea.Quit)
		return true

	case keyMsg.String() == "1":
		// 切换到Tab 1（项目）
		*cmds = append(*cmds, func() tea.Msg {
			return common.SwitchTabMsg{Index: 0}
		})
		return true

	case keyMsg.String() == "2":
		// 切换到Tab 2（任务）
		*cmds = append(*cmds, func() tea.Msg {
			return common.SwitchTabMsg{Index: 1}
		})
		return true

	case keyMsg.String() == "3":
		// 切换到Tab 3（搜索）
		*cmds = append(*cmds, func() tea.Msg {
			return common.SwitchTabMsg{Index: 2}
		})
		return true

	case keyMsg.String() == "4":
		// 切换到Tab 4（统计）
		*cmds = append(*cmds, func() tea.Msg {
			return common.SwitchTabMsg{Index: 3}
		})
		return true

	case keyMsg.String() == "ctrl+p":
		// Ctrl+P 显示/隐藏Command Palette
		if m.showCmdPalette {
			*cmds = append(*cmds, func() tea.Msg {
				return common.HideCommandPaletteMsg{}
			})
		} else {
			*cmds = append(*cmds, func() tea.Msg {
				return common.ShowCommandPaletteMsg{}
			})
		}
		return true
	}

	return false
}
