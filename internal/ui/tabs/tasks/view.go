package tasks

// View 渲染Tasks Tab
func (m Model) View() string {
	return m.taskList.View()
}
