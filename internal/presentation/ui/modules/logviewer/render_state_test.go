package logviewer

import (
	"testing"

	"github.com/aliancn/logcmd/internal/presentation/ui/common"
)

// TestRenderStateReset_OnFileSwitch 测试切换文件时渲染状态是否正确重置
func TestRenderStateReset_OnFileSwitch(t *testing.T) {
	theme := common.DefaultTheme()
	styles := common.DefaultStyles()
	viewer := New(theme, styles)

	// 模拟打开第一个文件并渲染
	viewer.SetFile("/path/to/file1.log")

	// 在第一次SetFile后，状态应该已被重置
	if viewer.lastRenderedHash != 0 {
		t.Errorf("第一次 SetFile() 没有重置 lastRenderedHash，期望 0，实际 %d", viewer.lastRenderedHash)
	}
	if !viewer.needsFullRender {
		t.Error("第一次 SetFile() 没有设置 needsFullRender = true")
	}

	// 模拟渲染完成后的状态
	viewer.lastRenderedHash = 12345
	viewer.needsFullRender = false

	// 切换到第二个文件
	viewer.SetFile("/path/to/file2.log")

	// 再次检查状态已重置
	if viewer.lastRenderedHash != 0 {
		t.Errorf("第二次 SetFile() 没有重置 lastRenderedHash，期望 0，实际 %d", viewer.lastRenderedHash)
	}
	if !viewer.needsFullRender {
		t.Error("第二次 SetFile() 没有设置 needsFullRender = true")
	}
}

// TestRenderStateReset_OnHistorySwitch 测试切换历史记录时渲染状态是否正确重置
func TestRenderStateReset_OnHistorySwitch(t *testing.T) {
	theme := common.DefaultTheme()
	styles := common.DefaultStyles()
	viewer := New(theme, styles)

	// 模拟已经渲染过
	viewer.lastRenderedHash = 11111
	viewer.needsFullRender = false

	// 切换历史记录（传nil模拟）
	viewer.SetHistory(nil)

	// 检查状态已重置
	if viewer.lastRenderedHash != 0 {
		t.Errorf("SetHistory() 没有重置 lastRenderedHash，期望 0，实际 %d", viewer.lastRenderedHash)
	}
	if !viewer.needsFullRender {
		t.Error("SetHistory() 没有设置 needsFullRender = true")
	}
}

// TestRenderStateReset_OnReset 测试Reset时渲染状态是否正确重置
func TestRenderStateReset_OnReset(t *testing.T) {
	theme := common.DefaultTheme()
	styles := common.DefaultStyles()
	viewer := New(theme, styles)

	// 模拟已经渲染过
	viewer.lastRenderedHash = 99999
	viewer.needsFullRender = false

	// 重置
	viewer.Reset()

	// 检查状态已重置
	if viewer.lastRenderedHash != 0 {
		t.Errorf("Reset() 没有重置 lastRenderedHash，期望 0，实际 %d", viewer.lastRenderedHash)
	}
	if !viewer.needsFullRender {
		t.Error("Reset() 没有设置 needsFullRender = true")
	}
}

// TestShouldRender_WithNeedsFullRender 测试needsFullRender标志强制渲染
func TestShouldRender_WithNeedsFullRender(t *testing.T) {
	theme := common.DefaultTheme()
	styles := common.DefaultStyles()
	viewer := New(theme, styles)

	// 设置相同的hash
	viewer.lastRenderedHash = 12345
	viewer.needsFullRender = true

	// 即使hash相同，也应该渲染
	if !viewer.shouldRender(12345) {
		t.Error("shouldRender() 应该在 needsFullRender=true 时返回 true，即使hash相同")
	}

	// 检查needsFullRender已被重置
	if viewer.needsFullRender {
		t.Error("shouldRender() 应该重置 needsFullRender = false")
	}
}

// TestShouldRender_WithSameHash 测试相同hash时跳过渲染
func TestShouldRender_WithSameHash(t *testing.T) {
	theme := common.DefaultTheme()
	styles := common.DefaultStyles()
	viewer := New(theme, styles)

	// 第一次渲染
	if !viewer.shouldRender(12345) {
		t.Error("shouldRender() 首次调用应该返回 true")
	}

	// 相同hash，应该跳过
	if viewer.shouldRender(12345) {
		t.Error("shouldRender() 相同hash时应该返回 false（跳过渲染）")
	}

	// 不同hash，应该渲染
	if !viewer.shouldRender(67890) {
		t.Error("shouldRender() 不同hash时应该返回 true")
	}
}
