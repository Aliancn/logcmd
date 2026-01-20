package logviewer

import (
	"os"
	"testing"

	"github.com/aliancn/logcmd/internal/presentation/ui/common"
)

// TestLoadSpecificFile 测试加载特定文件
func TestLoadSpecificFile(t *testing.T) {
	testFile := "/Users/aliancn/Downloads/tmp/.logcmd/2025-12-30/新配置_cat_2225.log"

	// 检查文件是否存在
	info, err := os.Stat(testFile)
	if err != nil {
		t.Skipf("测试文件不存在: %v", err)
	}

	t.Logf("✅ 文件存在: %s (%.2f MB)", testFile, float64(info.Size())/1024/1024)

	// 创建viewer
	theme := common.Theme{}
	styles := common.Styles{}
	viewer := New(theme, styles)
	viewer.SetSize(120, 40)

	// 设置文件
	t.Log("📂 调用 SetFile()...")
	viewer.SetFile(testFile)

	// 验证filePath已设置
	if viewer.filePath != testFile {
		t.Errorf("SetFile() 未正确设置 filePath。期望: %s, 实际: %s", testFile, viewer.filePath)
	}

	// 获取LoadContentCmd
	t.Log("📥 调用 LoadContentCmd()...")
	cmd := viewer.LoadContentCmd()

	if cmd == nil {
		t.Fatal("❌ LoadContentCmd() 返回 nil！")
	}

	// 执行命令
	t.Log("⚙️  执行 LoadContentCmd()...")
	msg := cmd()

	if msg == nil {
		t.Fatal("❌ 命令执行返回 nil 消息！")
	}

	t.Logf("✅ 收到消息类型: %T", msg)

	// 检查消息类型
	switch msg := msg.(type) {
	case ContentLoadedMsg:
		t.Logf("✅ ContentLoadedMsg: HistoryID=%d, ContentLen=%d bytes", msg.HistoryID, len(msg.Content))
		if msg.HistoryID != 0 {
			t.Errorf("期望 HistoryID=0（文件模式），实际: %d", msg.HistoryID)
		}
		if len(msg.Content) == 0 {
			t.Error("内容为空！")
		}
	case PartialContentLoadedMsg:
		t.Logf("✅ PartialContentLoadedMsg: HistoryID=%d, Lines=%d, IsFullFile=%v",
			msg.HistoryID, len(msg.Lines), msg.IsFullFile)
		if msg.HistoryID != 0 {
			t.Errorf("期望 HistoryID=0（文件模式），实际: %d", msg.HistoryID)
		}
		if len(msg.Lines) == 0 {
			t.Error("行数为空！")
		}
	case common.ErrorMsg:
		t.Fatalf("❌ ErrorMsg: %v", msg.Err)
	default:
		t.Fatalf("⚠️  未知消息类型: %T", msg)
	}

	// 模拟Update处理消息
	t.Log("📨 模拟 Update 处理消息...")
	updatedViewer, _ := viewer.Update(msg)

	// 检查loading状态
	if updatedViewer.loading {
		t.Error("❌ Update后 loading 仍为 true！")
	} else {
		t.Log("✅ loading 状态已更新为 false")
	}

	// 检查内容是否加载
	if updatedViewer.content == "" && len(updatedViewer.cachedLines) == 0 {
		t.Error("❌ 没有加载任何内容！")
	} else {
		t.Log("✅ 内容已加载")
	}
}
