package statspanel

import "github.com/aliancn/logcmd/internal/model"

// StatsLoadedMsg 统计数据加载完成
type StatsLoadedMsg struct {
	ProjectID   int
	CommandDist map[string]int
}

// SetProjectMsg 设置要显示的项目（来自 Projects Tab）
type SetProjectMsg struct {
	Project *model.Project
}
