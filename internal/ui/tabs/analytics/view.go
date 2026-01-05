package analytics

// View 渲染Analytics Tab
func (m Model) View() string {
	return m.statsPanel.View()
}
