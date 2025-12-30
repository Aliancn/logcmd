package executor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

// Result 记录命令执行结果
type Result struct {
	Command   string        // 执行的命令
	Args      []string      // 命令参数
	StartTime time.Time     // 开始时间
	EndTime   time.Time     // 结束时间
	Duration  time.Duration // 执行时长
	ExitCode  int           // 退出码
	Success   bool          // 是否成功
}

// Executor 命令执行器
type Executor struct {
	logFile io.Writer
	stdout  io.Writer
	stderr  io.Writer
	logMu   sync.Mutex
}

// New 创建新的执行器
func New(logFile io.Writer, stdout io.Writer, stderr io.Writer) *Executor {
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}

	return &Executor{
		logFile: logFile,
		stdout:  stdout,
		stderr:  stderr,
	}
}

// Execute 执行命令并记录输出
func (e *Executor) Execute(ctx context.Context, command string, args ...string) (*Result, error) {
	result := &Result{
		Command:   command,
		Args:      args,
		StartTime: time.Now(),
	}

	// 创建命令
	cmd := exec.CommandContext(ctx, command, args...)

	// 获取标准输出和错误输出的管道
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("获取stdout管道失败: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("获取stderr管道失败: %w", err)
	}

	// 启动命令
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("启动命令失败: %w", err)
	}

	// 使用WaitGroup等待所有输出处理完成
	var wg sync.WaitGroup
	wg.Add(2)

	// 处理标准输出
	go func() {
		defer wg.Done()
		e.streamOutput(stdoutPipe, e.stdout)
	}()

	// 处理标准错误
	go func() {
		defer wg.Done()
		e.streamOutput(stderrPipe, e.stderr)
	}()

	// 等待输出处理完成
	wg.Wait()

	// 等待命令完成
	err = cmd.Wait()
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// 获取退出码
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
		result.Success = false
	} else {
		result.ExitCode = 0
		result.Success = true
	}

	return result, nil
}

// streamOutput 流式处理输出，同时写入终端和日志文件
func (e *Executor) streamOutput(reader io.Reader, dest io.Writer) {
	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 256*1024)
	scanner.Buffer(buf, 1024*1024)

	var destWriter *bufio.Writer
	if dest != nil {
		destWriter = bufio.NewWriterSize(dest, 64*1024)
		defer func() {
			if err := destWriter.Flush(); err != nil {
				fmt.Fprintf(e.stderr, "刷新输出失败: %v\n", err)
			}
		}()
	}

	var logWriter *bufio.Writer
	if e.logFile != nil {
		logWriter = bufio.NewWriterSize(&lockedWriter{w: e.logFile, mu: &e.logMu}, 64*1024)
		defer func() {
			if err := logWriter.Flush(); err != nil {
				fmt.Fprintf(e.stderr, "刷新日志失败: %v\n", err)
			}
		}()
	}

	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)

		if destWriter != nil {
			if _, err := destWriter.Write(line); err != nil {
				fmt.Fprintf(e.stderr, "写入输出失败: %v\n", err)
			} else if err := destWriter.WriteByte('\n'); err != nil {
				fmt.Fprintf(e.stderr, "写入输出失败: %v\n", err)
			}
		}

		if logWriter != nil {
			if _, err := logWriter.Write(line); err != nil {
				fmt.Fprintf(e.stderr, "写入日志失败: %v\n", err)
			} else if err := logWriter.WriteByte('\n'); err != nil {
				fmt.Fprintf(e.stderr, "写入日志失败: %v\n", err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(e.stderr, "读取输出错误: %v\n", err)
	}
}

// WriteMetadata 写入命令元数据到日志
func (e *Executor) WriteMetadata(result *Result) {
	if e.logFile == nil {
		return
	}

	metadata := fmt.Sprintf(`
================================================================================
命令: %s %v
开始时间: %s
结束时间: %s
执行时长: %v
退出码: %d
执行状态: %s
================================================================================
`,
		result.Command,
		result.Args,
		result.StartTime.Format("2006-01-02 15:04:05"),
		result.EndTime.Format("2006-01-02 15:04:05"),
		result.Duration,
		result.ExitCode,
		map[bool]string{true: "成功", false: "失败"}[result.Success],
	)

	fmt.Fprint(e.logFile, metadata)
}

type lockedWriter struct {
	mu *sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}
