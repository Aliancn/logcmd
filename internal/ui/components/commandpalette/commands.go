package commandpalette

import "github.com/aliancn/logcmd/internal/ui/common"

// DefaultCommands 返回默认的命令列表
func DefaultCommands() []Command {
	return []Command{
		// Tab导航命令
		{
			ID:     "tab.projects",
			Label:  "转到项目管理",
			Desc:   "显示项目列表和历史记录 [Tab 1]",
			Action: common.SwitchTabMsg{Index: 0},
		},
		{
			ID:     "tab.tasks",
			Label:  "转到任务管理",
			Desc:   "查看后台运行的任务 [Tab 2]",
			Action: common.SwitchTabMsg{Index: 1},
		},
		{
			ID:     "tab.search",
			Label:  "全局搜索日志",
			Desc:   "在所有项目中搜索日志 [Tab 3]",
			Action: common.SwitchTabMsg{Index: 2},
		},
		{
			ID:     "tab.analytics",
			Label:  "统计分析",
			Desc:   "查看项目执行统计 [Tab 4]",
			Action: common.SwitchTabMsg{Index: 3},
		},

		// 快捷操作命令（可扩展）
		// {
		// 	ID:     "task.refresh",
		// 	Label:  "刷新任务列表",
		// 	Desc:   "重新加载所有后台任务",
		// 	Action: RefreshTasksMsg{},
		// },
		// {
		// 	ID:     "project.refresh",
		// 	Label:  "刷新项目列表",
		// 	Desc:   "重新加载所有项目",
		// 	Action: RefreshProjectsMsg{},
		// },
	}
}
