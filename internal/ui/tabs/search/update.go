package search

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Update 更新Search Tab
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	// 处理尺寸更新
	switch msg.(type) {
	case tea.WindowSizeMsg:
		m.searchView.SetSize(m.width, m.height)
	}

	// 将所有消息转发给searchView
	m.searchView, cmd = m.searchView.Update(msg)

	return m, cmd
}
