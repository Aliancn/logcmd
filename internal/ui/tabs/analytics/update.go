package analytics

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aliancn/logcmd/internal/ui/modules/statspanel"
)

// Update 更新Analytics Tab
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	// 处理尺寸更新
	switch msg.(type) {
	case tea.WindowSizeMsg:
		m.statsPanel.SetSize(m.width, m.height)
	}

	// 将所有消息转发给statsPanel
	var model tea.Model
	model, cmd = m.statsPanel.Update(msg)
	m.statsPanel = model.(statspanel.Model)

	return m, cmd
}
