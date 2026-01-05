package projects

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aliancn/logcmd/internal/ui/common"
	"github.com/aliancn/logcmd/internal/ui/modules/historylist"
	"github.com/aliancn/logcmd/internal/ui/modules/logviewer"
	"github.com/aliancn/logcmd/internal/ui/modules/projectlist"
	"github.com/aliancn/logcmd/internal/ui/modules/statspanel"
)

// Update 更新Projects Tab
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	// 处理模块间消息
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		// 尺寸更新由根Model处理，这里只需要分发给子模块
		// m.splitView.SetSize已经在 m.SetSize 中调用，这里也可以通过 Update 传递
		// 但由于我们在 Model.Update 中会调用 splitView.Update，所以这里可以省略或者作为备份
		// 为了一致性，我们在 SetSize 中处理了
	
	case projectlist.ProjectSelectedMsg:
		// 从项目列表切换到历史记录视图
		m.state = HistoryListView
		m.historyList.SetProject(typed.Project)
		cmds = append(cmds, m.historyList.LoadHistoryCmd())
		// 同时更新统计面板
		m.statsPanel.SetProject(typed.Project)
		cmds = append(cmds, m.statsPanel.LoadStatsCmd())
		// 更新面包屑
		cmds = append(cmds, func() tea.Msg {
			return common.UpdateBreadcrumbsMsg{Items: m.GetBreadcrumbs()}
		})

	case historylist.BackToProjectsMsg:
		// 从历史记录返回项目列表
		m.state = ProjectListView
		m.logViewer.Reset()
		// 更新面包屑
		cmds = append(cmds, func() tea.Msg {
			return common.UpdateBreadcrumbsMsg{Items: m.GetBreadcrumbs()}
		})

	case historylist.OpenLogMsg:
		// 从历史记录切换到日志查看
		m.state = LogViewerView
		m.logViewer.SetHistory(typed.History)
		cmds = append(cmds, m.logViewer.LoadContentCmd())
		// 更新面包屑
		cmds = append(cmds, func() tea.Msg {
			return common.UpdateBreadcrumbsMsg{Items: m.GetBreadcrumbs()}
		})

	case logviewer.BackMsg:
		// 从日志查看返回历史记录
		m.state = HistoryListView
		// 更新面包屑
		cmds = append(cmds, func() tea.Msg {
			return common.UpdateBreadcrumbsMsg{Items: m.GetBreadcrumbs()}
		})

	case projectlist.ProjectDeletedMsg:
		// 项目被删除，如果正在查看该项目的历史，返回项目列表
		if m.historyList.CurrentProjectID() == typed.ProjectID {
			m.state = ProjectListView
			m.historyList.SetProject(nil)
			m.logViewer.Reset()
		}
		// 重新加载项目列表
		cmds = append(cmds, m.projectList.LoadProjectsCmd())

	case statspanel.StatsLoadedMsg:
		// 统计数据加载完成，传递给statsPanel
		var model tea.Model
		model, cmd = m.statsPanel.Update(msg)
		m.statsPanel = model.(statspanel.Model)
		cmds = append(cmds, cmd)

	case common.ErrorMsg:
		// 错误消息传递给根Model处理
	}

	// 路由消息到当前子视图
	switch m.state {
	case ProjectListView:
		var model tea.Model
		
		// 1. 更新 ProjectList
		model, cmd = m.projectList.Update(msg)
		m.projectList = model.(projectlist.Model)
		cmds = append(cmds, cmd)
		
		// 2. 更新 StatsPanel
		model, cmd = m.statsPanel.Update(msg)
		m.statsPanel = model.(statspanel.Model)
		cmds = append(cmds, cmd)
		
		// 3. 同步最新的组件到 SplitView
		m.splitView.SetChildren(&m.projectList, &m.statsPanel)

	case HistoryListView:
		m.historyList, cmd = m.historyList.Update(msg)
		cmds = append(cmds, cmd)

	case LogViewerView:
		m.logViewer, cmd = m.logViewer.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}