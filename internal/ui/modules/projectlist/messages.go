package projectlist

import "github.com/aliancn/logcmd/internal/model"

// ProjectsLoadedMsg 项目列表加载完成
type ProjectsLoadedMsg struct {
	Projects []*model.Project
}

// ProjectSelectedMsg 项目被选中（传递给 Projects Tab）
type ProjectSelectedMsg struct {
	Project *model.Project
}

// ProjectDeletedMsg 项目被删除（传递给 Projects Tab）
type ProjectDeletedMsg struct {
	ProjectID int
}

// projectDeleteConfirmMsg 内部消息：开始删除确认流程
type projectDeleteConfirmMsg struct {
	project *model.Project
}
