package layout

import (
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

// Orientation 布局方向
type Orientation int

const (
	Horizontal Orientation = iota // 水平排列 (左右)
	Vertical                      // 垂直排列 (上下)
)

// SplitConfig 分割视图配置
type SplitConfig struct {
	// 基础配置
	Orientation     Orientation // 默认方向
	Ratio           float64     // 主视图占比 (0.0 - 1.0)
	SeparatorStyle  lipgloss.Style
	
	// 响应式配置
	MinWindowWidth  int  // 触发水平布局的最小窗口宽度
	HideSecondary   bool // 空间不足时是否隐藏次要视图
	MinSecondarySize int // 次要视图最小尺寸 (宽或高)
}

// DefaultSplitConfig 默认配置
func DefaultSplitConfig() SplitConfig {
	return SplitConfig{
		Orientation:      Horizontal,
		Ratio:            0.65, // 主视图占65%
		MinWindowWidth:   100,
		HideSecondary:    true,
		MinSecondarySize: 30,
	}
}

// SplitView 响应式分割视图
type SplitView struct {
	primary   Resizable
	secondary Resizable
	config    SplitConfig
	
	width  int
	height int
	
	// 当前实际计算出的状态
	currentOrientation Orientation
	secondaryVisible   bool
	primarySize        int // 宽或高，取决于方向
	secondarySize      int
}

// NewSplitView 创建新的分割视图
func NewSplitView(primary, secondary Resizable, config SplitConfig) *SplitView {
	return &SplitView{
		primary:   primary,
		secondary: secondary,
		config:    config,
	}
}

// SetChildren 更新子组件引用
// 在父组件 Update 返回新值后调用此方法
func (s *SplitView) SetChildren(primary, secondary Resizable) {
	s.primary = primary
	s.secondary = secondary
	// 重新计算尺寸，确保新组件获得正确的尺寸
	s.recalculate()
}

// Init 实现 tea.Model
func (s *SplitView) Init() tea.Cmd {
	return tea.Batch(s.primary.Init(), s.secondary.Init())
}

// Update 实现 tea.Model
func (s *SplitView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.SetSize(msg.Width, msg.Height)
	}

	// 更新子组件
	// 注意：这里需要类型断言更新回 Resizable 接口
	pModel, pCmd := s.primary.Update(msg)
	if p, ok := pModel.(Resizable); ok {
		s.primary = p
	}
	cmds = append(cmds, pCmd)

	if s.secondaryVisible {
		sModel, sCmd := s.secondary.Update(msg)
		if sec, ok := sModel.(Resizable); ok {
			s.secondary = sec
		}
		cmds = append(cmds, sCmd)
	}

	return s, tea.Batch(cmds...)
}

// SetSize 设置尺寸并重新计算布局
func (s *SplitView) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.recalculate()
}

// recalculate 核心布局逻辑
func (s *SplitView) recalculate() {
	if s.width == 0 || s.height == 0 {
		return
	}

	// 1. 决定是否隐藏次要视图
	// 如果配置了最小宽度且当前宽度不足，或者配置了总是隐藏
	if s.config.HideSecondary && s.width < s.config.MinWindowWidth {
		s.secondaryVisible = false
		s.currentOrientation = Vertical // 此时方向不重要，因为只有一个视图
		
		// 主视图占满全屏
		s.primary.SetSize(s.width, s.height)
		s.secondary.SetSize(0, 0)
		return
	}
	
	s.secondaryVisible = true
	s.currentOrientation = s.config.Orientation

	// 2. 根据方向计算尺寸
	if s.currentOrientation == Horizontal {
		// 水平布局：左右分割
		availWidth := s.width
		// 简单的按比例分配
		pWidth := int(float64(availWidth) * s.config.Ratio)
		sWidth := availWidth - pWidth
		
		// 检查次要视图最小尺寸约束
		if sWidth < s.config.MinSecondarySize {
			// 如果空间不足，优先保证最小尺寸，还是隐藏？
			// 这里选择调整比例
			sWidth = s.config.MinSecondarySize
			pWidth = availWidth - sWidth
		}
		
		// 传递尺寸给子组件 (高度占满)
		s.primary.SetSize(pWidth, s.height)
		s.secondary.SetSize(sWidth, s.height)
		
	} else {
		// 垂直布局：上下分割
		availHeight := s.height
		pHeight := int(float64(availHeight) * s.config.Ratio)
		sHeight := availHeight - pHeight
		
		if sHeight < s.config.MinSecondarySize {
			sHeight = s.config.MinSecondarySize
			pHeight = availHeight - sHeight
		}
		
		// 传递尺寸给子组件 (宽度占满)
		s.primary.SetSize(s.width, pHeight)
		s.secondary.SetSize(s.width, sHeight)
	}
}

// View 实现 tea.Model
func (s *SplitView) View() string {
	if !s.secondaryVisible {
		return s.primary.View()
	}

	pView := s.primary.View()
	sView := s.secondary.View()

	if s.currentOrientation == Horizontal {
		// 水平布局
		// 可以在这里添加分隔符逻辑
		return lipgloss.JoinHorizontal(lipgloss.Top, pView, sView)
	} else {
		// 垂直布局
		return lipgloss.JoinVertical(lipgloss.Left, pView, sView)
	}
}
