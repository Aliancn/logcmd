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
			if typed.Index == m.activeTabIndex {
				break
			}
			prev := m.activeTabIndex
			if cmd := m.handleTabDeactivated(prev); cmd != nil {
				cmds = append(cmds, cmd)
			}
			if typed.Index == 1 && prev != 1 { // Tasks is index 1
				m.lastNonTaskTabIndex = prev
			}
			m.activeTabIndex = typed.Index
			m.tabBar.SetActive(typed.Index)
			if typed.Index == 2 { // Search is index 2
				// 切换到搜索Tab时，尝试从项目列表获取当前选中的项目
				if p := m.projectsTab.SelectedProject(); p != nil {
					m.searchTab.SetCurrentProject(p)
				}
			}
			if cmd := m.handleTabActivated(typed.Index); cmd != nil {
				cmds = append(cmds, cmd)
			}
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

	case common.OpenProjectLogMsg:
		// Handle opening log from search results
		// Switch to Projects Tab (Index 0)
		if m.activeTabIndex != 0 {
			// Deactivate previous tab if needed
			if m.activeTabIndex == 2 { // Search is index 2
				m.searchTab.OnDeactivated()
			}

			m.activeTabIndex = 0
			m.tabBar.SetActive(0)

			// Activate new tab
			// m.projectsTab.OnActivated() // If needed
		}
		// Pass the message to the projects tab
		var cmd tea.Cmd
		m.projectsTab, cmd = m.projectsTab.Update(typed)
		cmds = append(cmds, cmd)

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
	case key.Matches(keyMsg, m.globalKeys.Back):
		if m.handleTaskBack(cmds) {
			return true
		}
		return false

	case key.Matches(keyMsg, m.globalKeys.Search):
		if m.showCmdPalette {
			return false
		}
		if m.activeTabIndex == 2 { // Search is index 2
			if cmd := m.searchTab.FocusInput(); cmd != nil {
				*cmds = append(*cmds, cmd)
			}
		} else {
			*cmds = append(*cmds, func() tea.Msg {
				return common.SwitchTabMsg{Index: 2}
			})
		}
		return true

	case key.Matches(keyMsg, m.globalKeys.Quit):
		// 确保 Esc 不会触发退出（即使绑定匹配）
		if keyMsg.Type == tea.KeyEsc || keyMsg.String() == "esc" {
			return false
		}

		// Don't quit if 'q' is pressed while search input is focused
		if keyMsg.String() == "q" && m.activeTabIndex == 2 && m.searchTab.IsInputFocused() {
			return false
		}

		// Ctrl+C/q 退出
		*cmds = append(*cmds, tea.Quit)
		return true

	case keyMsg.String() == "1":
		// 切换到Tab 1
		*cmds = append(*cmds, func() tea.Msg {
			return common.SwitchTabMsg{Index: 0}
		})
		return true

	case keyMsg.String() == "2":
		// 切换到Tab 2
		*cmds = append(*cmds, func() tea.Msg {
			return common.SwitchTabMsg{Index: 1}
		})
		return true

	case keyMsg.String() == "3":
		// 切换到Tab 3
		*cmds = append(*cmds, func() tea.Msg {
			return common.SwitchTabMsg{Index: 2}
		})
		return true

	case keyMsg.String() == "4":
		// 切换到Tab 4
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
