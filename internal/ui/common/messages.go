package common

// ErrorMsg 用于跨模块报告错误。
type ErrorMsg struct {
	Err error
}

// KeyHint 快捷键提示（从footer包移到这里避免循环导入）
type KeyHint struct {
	Key  string // 快捷键（如"Ctrl+C"）
	Desc string // 描述（如"退出"）
}

// Tab导航相关消息

// SwitchTabMsg Tab切换消息
type SwitchTabMsg struct {
	Index int // 目标Tab索引（0-3）
}

// UpdateBreadcrumbsMsg 面包屑更新消息
type UpdateBreadcrumbsMsg struct {
	Items []string // 面包屑路径项
}

// Command Palette相关消息

// ShowCommandPaletteMsg 显示Command Palette
type ShowCommandPaletteMsg struct{}

// HideCommandPaletteMsg 隐藏Command Palette
type HideCommandPaletteMsg struct{}

// ExecuteCommandMsg 执行命令
type ExecuteCommandMsg struct {
	CommandID string // 命令ID
}

// OpenProjectLogMsg 打开特定项目的日志文件
type OpenProjectLogMsg struct {
	Project  interface{} // 使用interface{}避免循环导入，实际为 *model.Project
	FilePath string
	LineNum  int
}
