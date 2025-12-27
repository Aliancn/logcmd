package model_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/aliancn/logcmd/internal/model"
)

func TestCommandHistory_BeforeSave(t *testing.T) {
	cmd := &model.CommandHistory{
		Command:     "git commit -m test",
		CommandArgs: []string{"commit", "-m", "test"},
		ExitCode:    0,
	}

	err := cmd.BeforeSave()
	if err != nil {
		t.Fatalf("BeforeSave() 失败: %v", err)
	}

	// 验证 CommandArgs 被序列化
	if cmd.ArgsJSON == "" {
		t.Error("ArgsJSON 不应为空")
	}

	var args []string
	err = json.Unmarshal([]byte(cmd.ArgsJSON), &args)
	if err != nil {
		t.Fatalf("反序列化 ArgsJSON 失败: %v", err)
	}

	if len(args) != len(cmd.CommandArgs) {
		t.Errorf("参数数量不匹配: got %d, want %d", len(args), len(cmd.CommandArgs))
	}

	// 验证命令名称被提取
	if cmd.CommandName == "" {
		t.Error("CommandName 应该被提取出来")
	}

	// 验证状态被设置
	if cmd.Status != "success" {
		t.Errorf("Status 应该是 success: got %s", cmd.Status)
	}
}

func TestCommandHistory_BeforeSaveFailedStatus(t *testing.T) {
	cmd := &model.CommandHistory{
		Command:  "failing_command",
		ExitCode: 1,
	}

	err := cmd.BeforeSave()
	if err != nil {
		t.Fatalf("BeforeSave() 失败: %v", err)
	}

	if cmd.Status != "failed" {
		t.Errorf("ExitCode 非零时 Status 应该是 failed: got %s", cmd.Status)
	}
}

func TestCommandHistory_AfterLoad(t *testing.T) {
	argsJSON := `["commit", "-m", "test message"]`

	cmd := &model.CommandHistory{
		ArgsJSON: argsJSON,
	}

	err := cmd.AfterLoad()
	if err != nil {
		t.Fatalf("AfterLoad() 失败: %v", err)
	}

	expectedArgs := []string{"commit", "-m", "test message"}
	if len(cmd.CommandArgs) != len(expectedArgs) {
		t.Fatalf("参数数量不匹配: got %d, want %d", len(cmd.CommandArgs), len(expectedArgs))
	}

	for i, arg := range cmd.CommandArgs {
		if arg != expectedArgs[i] {
			t.Errorf("参数 %d 不匹配: got %s, want %s", i, arg, expectedArgs[i])
		}
	}
}

func TestCommandHistory_AfterLoadInvalidJSON(t *testing.T) {
	cmd := &model.CommandHistory{
		ArgsJSON: "invalid json",
	}

	err := cmd.AfterLoad()
	if err == nil {
		t.Error("AfterLoad() 应该对无效 JSON 返回错误")
	}
}

func TestCommandHistory_GetDuration(t *testing.T) {
	tests := []struct {
		name       string
		durationMs int64
		expected   time.Duration
	}{
		{"零时长", 0, 0},
		{"一秒", 1000, time.Second},
		{"半秒", 500, 500 * time.Millisecond},
		{"十秒", 10000, 10 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &model.CommandHistory{
				DurationMs: tt.durationMs,
			}

			duration := cmd.GetDuration()
			if duration != tt.expected {
				t.Errorf("GetDuration() = %v, want %v", duration, tt.expected)
			}
		})
	}
}

func TestCommandHistory_IsSuccess(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"成功状态", "success", true},
		{"失败状态", "failed", false},
		{"空状态", "", false},
		{"未知状态", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &model.CommandHistory{
				Status: tt.status,
			}

			if got := cmd.IsSuccess(); got != tt.expected {
				t.Errorf("IsSuccess() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTruncateOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "短文本不截断",
			input:    "short",
			maxLen:   10,
			expected: "short",
		},
		{
			name:     "长文本截断",
			input:    "this is a very long text that should be truncated",
			maxLen:   10,
			expected: "this is a ...",
		},
		{
			name:     "恰好等于长度",
			input:    "exact",
			maxLen:   5,
			expected: "exact",
		},
		{
			name:     "空字符串",
			input:    "",
			maxLen:   10,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := model.TruncateOutput(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("TruncateOutput() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestCommandHistory_FullWorkflow(t *testing.T) {
	// 创建一个完整的命令历史记录
	startTime := time.Now()
	endTime := startTime.Add(2 * time.Second)

	cmd := &model.CommandHistory{
		ProjectID:        1,
		Command:          "go test -v",
		CommandArgs:      []string{"test", "-v"},
		StartTime:        startTime,
		EndTime:          endTime,
		DurationMs:       2000,
		ExitCode:         0,
		LogFilePath:      "/path/to/log.log",
		LogDate:          "2024-01-01",
		StdoutPreview:    "ok",
		StderrPreview:    "",
		HasError:         false,
		WorkingDirectory: "/home/user/project",
		EnvironmentJSON:  `{"PATH":"/usr/bin"}`,
		CreatedAt:        time.Now(),
	}

	// 测试 BeforeSave
	err := cmd.BeforeSave()
	if err != nil {
		t.Fatalf("BeforeSave() 失败: %v", err)
	}

	// 验证字段
	if cmd.CommandName != "go" {
		t.Errorf("CommandName = %s, want go", cmd.CommandName)
	}

	if cmd.Status != "success" {
		t.Errorf("Status = %s, want success", cmd.Status)
	}

	if cmd.ArgsJSON == "" {
		t.Error("ArgsJSON 不应为空")
	}

	// 模拟从数据库加载（清空运行时字段）
	savedArgsJSON := cmd.ArgsJSON
	cmd.CommandArgs = nil

	// 测试 AfterLoad
	cmd.ArgsJSON = savedArgsJSON
	err = cmd.AfterLoad()
	if err != nil {
		t.Fatalf("AfterLoad() 失败: %v", err)
	}

	// 验证数据恢复
	if len(cmd.CommandArgs) != 2 {
		t.Errorf("CommandArgs 长度 = %d, want 2", len(cmd.CommandArgs))
	}

	// 测试辅助方法
	if !cmd.IsSuccess() {
		t.Error("IsSuccess() 应该返回 true")
	}

	duration := cmd.GetDuration()
	if duration != 2*time.Second {
		t.Errorf("GetDuration() = %v, want 2s", duration)
	}
}
