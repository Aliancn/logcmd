package projects

// View 渲染Projects Tab
func (m Model) View() string {
	// 根据当前子状态渲染不同视图
	switch m.state {
	case ProjectListView:
		return m.renderProjectListWithStats()
	case HistoryListView:
		return m.historyList.View()
	case LogViewerView:
		return m.logViewer.View()
	default:
		return m.styles.Error.Render("未知项目视图状态")
	}
}

// renderProjectListWithStats 渲染项目列表
func (m Model) renderProjectListWithStats() string {
	return m.projectList.View()
}
