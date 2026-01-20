package common

// 布局常量定义

const (
	// 三段式布局高度分配
	// HeaderHeight  = 4 // Header Removed
	TabBarHeight  = 2 // TabBar含标签行与下划线
	FooterHeight  = 1 // Footer状态栏行数
	TotalOverhead = TabBarHeight + FooterHeight

	// 响应式布局阈值
	MinWidthForStats      = 90  // 最小宽度才显示统计面板
	MinWidthForHorizontal = 130 // 最小宽度才使用水平布局
	MinListWidth          = 50  // 列表最小宽度
	MinStatsWidth         = 35  // 统计面板最小宽度
	SpacerWidth           = 1   // 水平布局时两个面板之间的间距

	// Projects Tab布局比例
	ProjectsListHeightRatio  = 60 // 垂直布局时列表占比（%）
	ProjectsListWidthRatio   = 65 // 水平布局时列表占比（%）
	ProjectsStatsWidthRatio  = 35 // 水平布局时统计面板占比（%）
	ProjectsStatsMaxWidthPct = 40 // 统计面板最大宽度占比（%）

	// 最小尺寸保证
	MinMainAreaHeight  = 10 // Main区域最小高度
	MinSplitHeight     = 20 // 最小高度才能进行垂直分割
	MinComponentHeight = 8  // 单个组件最小高度

	// Command Palette尺寸
	CommandPaletteMaxWidth    = 80 // Command Palette最大宽度
	CommandPaletteMaxHeight   = 20 // Command Palette最大高度
	CommandPaletteWidthRatio  = 66 // Command Palette宽度占比（%）
	CommandPaletteHeightRatio = 50 // Command Palette高度占比（%）
)
