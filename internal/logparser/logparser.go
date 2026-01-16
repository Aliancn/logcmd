package logparser

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	maxHeaderScanLines = 32
	logFooterReadSize  = 16 * 1024
)

var (
	headerDateRegex    = regexp.MustCompile(`^# 时间:\s*(.+)$`)
	headerCommandRegex = regexp.MustCompile(`^# 命令:\s*(.+)$`)
	footerCommandRegex = regexp.MustCompile(`^命令:\s*(.+)$`)
	startTimeRegex     = regexp.MustCompile(`^开始时间:\s*(.+)$`)
	endTimeRegex       = regexp.MustCompile(`^结束时间:\s*(.+)$`)
	durationRegex      = regexp.MustCompile(`^执行时长:\s*(.+)$`)
	exitCodeRegex      = regexp.MustCompile(`^退出码:\s*(\d+)$`)
	statusRegex        = regexp.MustCompile(`^执行状态:\s*(\S+)$`)
)

// Metadata 表示从日志文件解析出的结构化信息。
type Metadata struct {
	CommandName       string
	CommandLine       string
	CommandArgs       []string
	CommandFromFooter bool

	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration

	ExitCode    int
	ExitCodeSet bool

	StatusText string
	StatusSet  bool
	Success    bool

	LogDate    string
	HeaderTime time.Time
}

// ParseFile 解析单个日志文件的元数据。
func ParseFile(ctx context.Context, path string) (*Metadata, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	meta := &Metadata{}
	if err := parseHeader(ctx, file, meta); err != nil {
		return nil, err
	}
	if err := parseFooter(ctx, file, meta); err != nil {
		return nil, err
	}

	if err := finalizeMetadata(meta); err != nil {
		return nil, err
	}
	return meta, nil
}

func parseHeader(ctx context.Context, file io.Reader, meta *Metadata) error {
	if seeker, ok := file.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return err
		}
	}

	scanner := bufio.NewScanner(file)
	lines := 0
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}

		line := scanner.Text()
		lines++

		if meta.HeaderTime.IsZero() {
			if matches := headerDateRegex.FindStringSubmatch(line); matches != nil {
				if t, err := parseTime(matches[1]); err == nil {
					meta.HeaderTime = t
					if meta.LogDate == "" {
						meta.LogDate = t.Format("2006-01-02")
					}
				}
			}
		}

		if meta.CommandName == "" {
			if matches := headerCommandRegex.FindStringSubmatch(line); matches != nil {
				setCommandMeta(meta, matches[1], false)
			}
		}

		if lines >= maxHeaderScanLines {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return ctx.Err()
}

func parseFooter(ctx context.Context, file *os.File, meta *Metadata) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}

	size := info.Size()
	if size == 0 {
		return nil
	}

	readSize := logFooterReadSize
	if int64(readSize) > size {
		readSize = int(size)
	}
	start := size - int64(readSize)
	buf := make([]byte, readSize)
	if err := ctx.Err(); err != nil {
		return err
	}

	if _, err := file.ReadAt(buf, start); err != nil && err != io.EOF {
		return err
	}

	lines := bytes.Split(buf, []byte{'\n'})
	for _, line := range lines {
		lineStr := string(bytes.TrimSpace(line))
		if lineStr == "" {
			continue
		}

		if matches := footerCommandRegex.FindStringSubmatch(lineStr); matches != nil {
			setCommandMeta(meta, matches[1], true)
			continue
		}

		if matches := startTimeRegex.FindStringSubmatch(lineStr); matches != nil {
			if t, err := parseTime(matches[1]); err == nil {
				meta.StartTime = t
			}
			continue
		}

		if matches := endTimeRegex.FindStringSubmatch(lineStr); matches != nil {
			if t, err := parseTime(matches[1]); err == nil {
				meta.EndTime = t
			}
			continue
		}

		if matches := durationRegex.FindStringSubmatch(lineStr); matches != nil {
			meta.Duration = parseDuration(matches[1])
			continue
		}

		if matches := exitCodeRegex.FindStringSubmatch(lineStr); matches != nil {
			fmt.Sscanf(matches[1], "%d", &meta.ExitCode)
			meta.ExitCodeSet = true
			continue
		}

		if matches := statusRegex.FindStringSubmatch(lineStr); matches != nil {
			meta.StatusText = matches[1]
			meta.StatusSet = true
			continue
		}
	}

	return nil
}

func finalizeMetadata(meta *Metadata) error {
	if meta.CommandLine == "" && meta.CommandName != "" {
		meta.CommandLine = buildCommandLine(meta.CommandName, meta.CommandArgs)
	}

	if meta.StartTime.IsZero() && !meta.HeaderTime.IsZero() {
		meta.StartTime = meta.HeaderTime
	}

	if meta.Duration == 0 && !meta.StartTime.IsZero() && !meta.EndTime.IsZero() {
		meta.Duration = meta.EndTime.Sub(meta.StartTime)
	}

	if meta.EndTime.IsZero() && !meta.StartTime.IsZero() && meta.Duration > 0 {
		meta.EndTime = meta.StartTime.Add(meta.Duration)
	}

	if meta.StatusSet {
		meta.Success = meta.StatusText == "成功"
	} else if meta.ExitCodeSet {
		meta.Success = meta.ExitCode == 0
	}

	if meta.Duration < 0 {
		meta.Duration = 0
	}

	return nil
}

func setCommandMeta(meta *Metadata, raw string, fromFooter bool) {
	command, args := parseCommandLine(raw)
	if command == "" {
		return
	}
	meta.CommandName = command
	meta.CommandArgs = args
	meta.CommandLine = buildCommandLine(command, args)
	if fromFooter {
		meta.CommandFromFooter = true
	}
}

func parseCommandLine(raw string) (string, []string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}

	if idx := strings.Index(raw, "["); idx != -1 && strings.HasSuffix(raw, "]") {
		command := strings.TrimSpace(raw[:idx])
		argsPart := strings.TrimSpace(strings.TrimSuffix(raw[idx:], "]"))
		argsPart = strings.TrimPrefix(argsPart, "[")
		if argsPart == "" {
			return command, nil
		}
		return command, strings.Fields(argsPart)
	}

	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return "", nil
	}
	if len(parts) == 1 {
		return parts[0], nil
	}
	return parts[0], parts[1:]
}

func buildCommandLine(command string, args []string) string {
	if command == "" {
		return ""
	}
	if len(args) == 0 {
		return command
	}
	parts := append([]string{command}, args...)
	return strings.Join(parts, " ")
}

func parseDuration(raw string) time.Duration {
	raw = strings.ReplaceAll(raw, " ", "")
	duration, _ := time.ParseDuration(raw)
	return duration
}

func parseTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("空时间")
	}
	return time.ParseInLocation("2006-01-02 15:04:05", raw, time.Local)
}
