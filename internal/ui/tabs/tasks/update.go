package tasks

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Update 更新Tasks Tab
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	// 处理尺寸更新
	switch msg.(type) {
	case tea.WindowSizeMsg:
		m.taskList.SetSize(m.width, m.height)
	}

	// 将所有消息转发给taskList
	m.taskList, cmd = m.taskList.Update(msg)

	return m, cmd
}
