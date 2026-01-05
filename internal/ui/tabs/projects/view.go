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

// renderProjectListWithStats 渲染项目列表和统计面板
func (m Model) renderProjectListWithStats() string {
	// 使用 SplitView 渲染，它会自动处理布局和隐藏逻辑
	// 注意：我们需要确保 SplitView 引用的 children 是最新的。
	// 由于我们在 update.go 中更新了 m.projectList 和 m.statsPanel，
	// 如果 SplitView 持有的是指针，那么它们需要指向同一个地址。
	// 在 New 函数中我们传的是 &pList, &sPanel。
	// 但在 Update 中 m.projectList = model.(projectlist.Model) 
	// 这会改变 m.projectList 的值，但不会改变 New 函数里那个局部变量的地址。
	// 实际上，Model 结构体里的 projectList 是值类型。
	// New 函数里 &pList 取的是局部变量的地址，这是很危险的 (虽然 Go 会逃逸分析，但 Update 时 m.projectList 已经是新的副本了)。
	
	// 所以，最稳妥的方式是：
	// 每次 Render 前 (或者 Update 后)，更新 SplitView 的 children。
	// 由于 SplitView 结构体没有 SetChildren 方法，我们这里通过 SetSize 间接更新? 不行。
	// 
	// 简单粗暴：view 时创建一个临时的 SplitView ? 
	// 不行，状态丢失。
	
	// 正确的做法是：给 SplitView 加 SetChildren 方法，并在 View() 中调用?
	// 不，View 应该是无副作用的。
	
	// 我们需要在 Update 中调用 m.splitView.SetChildren(&m.projectList, &m.statsPanel)
	// 我会在下一步给 split.go 加这个方法。
	
	return m.splitView.View()
}
