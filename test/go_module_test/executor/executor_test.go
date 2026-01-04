package executor_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/aliancn/logcmd/internal/executor"
)

func newTestExecutor(logFile io.Writer) *executor.Executor {
	return executor.New(logFile, io.Discard, io.Discard)
}

func TestNew(t *testing.T) {
	var buf bytes.Buffer
	exec := newTestExecutor(&buf)

	if exec == nil {
		t.Fatal("New() 返回了 nil")
	}
}

func TestExecute_Success(t *testing.T) {
	var buf bytes.Buffer
	exec := newTestExecutor(&buf)

	ctx := context.Background()
	result, err := exec.Execute(ctx, "echo", "hello", "world")

	if err != nil {
		t.Fatalf("Execute() 失败: %v", err)
	}

	if result == nil {
		t.Fatal("result 不应为 nil")
	}

	// 验证执行结果
	if result.Command != "echo" {
		t.Errorf("Command = %s, want echo", result.Command)
	}

	if len(result.Args) != 2 {
		t.Errorf("Args 长度 = %d, want 2", len(result.Args))
	}

	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}

	if !result.Success {
		t.Error("Success 应该为 true")
	}

	// 验证时间相关字段
	if result.StartTime.IsZero() {
		t.Error("StartTime 不应为零值")
	}

	if result.EndTime.IsZero() {
		t.Error("EndTime 不应为零值")
	}

	if result.Duration <= 0 {
		t.Error("Duration 应该大于 0")
	}

	if result.EndTime.Before(result.StartTime) {
		t.Error("EndTime 应该在 StartTime 之后")
	}
}

func TestExecute_Failure(t *testing.T) {
	var buf bytes.Buffer
	exec := newTestExecutor(&buf)

	ctx := context.Background()
	// 执行一个会失败的命令
	result, err := exec.Execute(ctx, "false")

	// Execute 不应返回错误，而是通过 result 表示失败
	if err != nil {
		t.Logf("Execute() 返回错误: %v", err)
	}

	if result == nil {
		t.Fatal("result 不应为 nil")
	}

	if result.Success {
		t.Error("Success 应该为 false")
	}

	if result.ExitCode == 0 {
		t.Error("ExitCode 应该非零")
	}
}

func TestExecute_NonExistentCommand(t *testing.T) {
	var buf bytes.Buffer
	exec := newTestExecutor(&buf)

	ctx := context.Background()
	result, err := exec.Execute(ctx, "nonexistent_command_12345")

	// 不存在的命令应该返回错误
	if err == nil {
		t.Error("Execute() 应该对不存在的命令返回错误")
	}

	// result 可能为 nil，也可能包含错误信息
	if result != nil {
		if result.Success {
			t.Error("不存在的命令不应标记为成功")
		}
	}
}

func TestExecute_WithOutput(t *testing.T) {
	var buf bytes.Buffer
	exec := newTestExecutor(&buf)

	ctx := context.Background()
	testString := "test output message"
	result, err := exec.Execute(ctx, "echo", testString)

	if err != nil {
		t.Fatalf("Execute() 失败: %v", err)
	}

	if !result.Success {
		t.Error("命令应该成功执行")
	}

	// 验证日志输出包含命令输出本身
	logContent := buf.String()
	if !strings.Contains(logContent, testString) {
		t.Errorf("日志应该包含命令输出: %s", testString)
	}
}

func TestExecute_WithContext(t *testing.T) {
	var buf bytes.Buffer
	exec := newTestExecutor(&buf)

	// 创建一个会超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// 执行一个长时间运行的命令
	result, err := exec.Execute(ctx, "sleep", "10")

	// 应该返回错误或者标记为失败
	if err == nil && (result == nil || result.Success) {
		t.Error("超时的命令应该失败")
	}

	if result != nil {
		if result.Success {
			t.Error("超时的命令不应标记为成功")
		}
	}
}

func TestWriteMetadata(t *testing.T) {
	var buf bytes.Buffer
	exec := newTestExecutor(&buf)

	startTime := time.Now()
	result := &executor.Result{
		Command:   "test",
		Args:      []string{"arg1", "arg2"},
		StartTime: startTime,
		EndTime:   startTime.Add(time.Second),
		Duration:  time.Second,
		ExitCode:  0,
		Success:   true,
	}

	exec.WriteMetadata(result)

	metadata := buf.String()

	// 验证元数据包含必要信息
	if !strings.Contains(metadata, "test") {
		t.Error("元数据应该包含命令名称")
	}

	if !strings.Contains(metadata, "开始时间") {
		t.Error("元数据应该包含开始时间")
	}

	if !strings.Contains(metadata, "结束时间") {
		t.Error("元数据应该包含结束时间")
	}

	if !strings.Contains(metadata, "执行时长") {
		t.Error("元数据应该包含执行时长")
	}

	if !strings.Contains(metadata, "退出码") {
		t.Error("元数据应该包含退出码")
	}

	if !strings.Contains(metadata, "执行状态") {
		t.Error("元数据应该包含执行状态")
	}

	if !strings.Contains(metadata, "成功") {
		t.Error("成功的命令应该显示'成功'状态")
	}
}

func TestWriteMetadataWithNilLogFile(t *testing.T) {
	// 创建没有日志文件的执行器
	exec := newTestExecutor(nil)

	result := &executor.Result{
		Command:   "test",
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Duration:  0,
		ExitCode:  0,
		Success:   true,
	}

	// 不应该 panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("WriteMetadata() panic: %v", r)
		}
	}()

	exec.WriteMetadata(result)
}

func TestExecute_MultipleCommands(t *testing.T) {
	commands := []struct {
		cmd  string
		args []string
	}{
		{"echo", []string{"test1"}},
		{"echo", []string{"test2"}},
		{"pwd", []string{}},
	}

	for _, tc := range commands {
		t.Run(tc.cmd, func(t *testing.T) {
			var buf bytes.Buffer
			exec := newTestExecutor(&buf)

			ctx := context.Background()
			result, err := exec.Execute(ctx, tc.cmd, tc.args...)

			if err != nil {
				t.Fatalf("Execute(%s) 失败: %v", tc.cmd, err)
			}

			if !result.Success {
				t.Errorf("Execute(%s) 应该成功", tc.cmd)
			}
		})
	}
}

func TestResult_Fields(t *testing.T) {
	result := &executor.Result{
		Command:   "test",
		Args:      []string{"arg1"},
		StartTime: time.Now(),
		EndTime:   time.Now().Add(time.Second),
		Duration:  time.Second,
		ExitCode:  0,
		Success:   true,
	}

	// 验证所有字段都被正确设置
	if result.Command != "test" {
		t.Errorf("Command = %s, want test", result.Command)
	}

	if len(result.Args) != 1 {
		t.Errorf("Args 长度 = %d, want 1", len(result.Args))
	}

	if result.Duration != time.Second {
		t.Errorf("Duration = %v, want 1s", result.Duration)
	}

	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}

	if !result.Success {
		t.Error("Success 应该为 true")
	}
}
