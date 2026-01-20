package logviewer

import (
	"testing"

	"github.com/aliancn/logcmd/internal/model"
	"github.com/aliancn/logcmd/internal/ui/common"
)

// TestSetFile_SetsFilePath 确保SetFile正确设置filePath字段
func TestSetFile_SetsFilePath(t *testing.T) {
	theme := common.Theme{}
	styles := common.Styles{}
	m := New(theme, styles)

	testPath := "/test/path/to/file.log"
	m.SetFile(testPath)

	if m.filePath != testPath {
		t.Errorf("SetFile() 未正确设置 filePath。期望: %s, 实际: %s", testPath, m.filePath)
	}

	if m.history != nil {
		t.Error("SetFile() 应该清除 history")
	}

	if m.loading != true {
		t.Error("SetFile() 应该设置 loading = true")
	}

	if m.highlightCache == nil {
		t.Error("SetFile() 应该初始化 highlightCache")
	}
}

// TestSetHistory_ClearsFilePath 确保SetHistory清除filePath字段
func TestSetHistory_ClearsFilePath(t *testing.T) {
	theme := common.Theme{}
	styles := common.Styles{}
	m := New(theme, styles)

	// 先设置一个文件路径
	m.filePath = "/some/file.log"

	// 然后设置历史记录
	history := &model.CommandHistory{
		ID:          1,
		Command:     "test",
		LogFilePath: "/history/log.log",
	}
	m.SetHistory(history)

	if m.filePath != "" {
		t.Errorf("SetHistory() 应该清除 filePath。期望: 空字符串, 实际: %s", m.filePath)
	}

	if m.history != history {
		t.Error("SetHistory() 应该设置 history")
	}

	if m.loading != true {
		t.Error("SetHistory() 应该设置 loading = true")
	}

	if m.highlightCache == nil {
		t.Error("SetHistory() 应该初始化 highlightCache")
	}
}

// TestLoadContentCmd_WithFilePath 测试LoadContentCmd能够使用filePath
func TestLoadContentCmd_WithFilePath(t *testing.T) {
	theme := common.Theme{}
	styles := common.Styles{}
	m := New(theme, styles)

	testPath := "/test/path/to/file.log"
	m.SetFile(testPath)

	cmd := m.LoadContentCmd()
	if cmd == nil {
		t.Error("LoadContentCmd() 不应该返回 nil，当 filePath 已设置")
	}
}

// TestLoadContentCmd_WithHistory 测试LoadContentCmd能够使用history
func TestLoadContentCmd_WithHistory(t *testing.T) {
	theme := common.Theme{}
	styles := common.Styles{}
	m := New(theme, styles)

	history := &model.CommandHistory{
		ID:          1,
		Command:     "test",
		LogFilePath: "/history/log.log",
	}
	m.SetHistory(history)

	cmd := m.LoadContentCmd()
	if cmd == nil {
		t.Error("LoadContentCmd() 不应该返回 nil，当 history 已设置")
	}
}

// TestLoadContentCmd_WithoutPath 测试LoadContentCmd在没有路径时返回nil
func TestLoadContentCmd_WithoutPath(t *testing.T) {
	theme := common.Theme{}
	styles := common.Styles{}
	m := New(theme, styles)

	// 不设置任何路径或历史记录
	cmd := m.LoadContentCmd()
	if cmd != nil {
		t.Error("LoadContentCmd() 应该返回 nil，当没有路径或历史记录时")
	}
}
