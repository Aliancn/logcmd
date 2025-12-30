package search

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/aliancn/logcmd/internal/walker"
)

// SearchOptions 搜索选项
type SearchOptions struct {
	LogDir        string    // 日志目录
	Keyword       string    // 搜索关键词
	UseRegex      bool      // 使用正则表达式
	StartDate     time.Time // 开始日期
	EndDate       time.Time // 结束日期
	CaseSensitive bool      // 区分大小写
	ShowContext   int       // 显示上下文行数
	CompiledRegex *regexp.Regexp
}

// SearchResult 搜索结果
type SearchResult struct {
	FilePath string   // 文件路径
	LineNum  int      // 行号
	Line     string   // 匹配的行
	Context  []string // 上下文行
}

// Searcher 日志搜索器
type Searcher struct {
	options         *SearchOptions
	regex           *regexp.Regexp
	lowerKeyword    string
	useASCIIMatcher bool
	asciiKeyword    []byte
}

// ResultHandler 处理搜索结果
type ResultHandler func(*SearchResult) error

type pendingContext struct {
	result    *SearchResult
	remaining int
}

// New 创建搜索器
func New(options *SearchOptions) (*Searcher, error) {
	s := &Searcher{
		options: options,
	}

	// 如果使用正则表达式，编译它
	if options.UseRegex {
		if options.CompiledRegex != nil {
			s.regex = options.CompiledRegex
		} else {
			flags := ""
			if !options.CaseSensitive {
				flags = "(?i)"
			}
			regex, err := regexp.Compile(flags + options.Keyword)
			if err != nil {
				return nil, fmt.Errorf("正则表达式编译失败: %w", err)
			}
			s.regex = regex
		}
	} else if !options.CaseSensitive {
		lower := strings.ToLower(options.Keyword)
		s.lowerKeyword = lower
		if isASCII(lower) {
			s.useASCIIMatcher = true
			s.asciiKeyword = []byte(lower)
		}
	}

	return s, nil
}

// Search 执行搜索，并在每次匹配时调用 handler
func (s *Searcher) Search(ctx context.Context, handler ResultHandler) error {
	if handler == nil {
		return errors.New("handler 不能为空")
	}

	fileWalker, err := walker.New(walker.Options{
		Root: s.options.LogDir,
		FileFilter: func(path string, info os.FileInfo) bool {
			if !strings.HasSuffix(path, ".log") {
				return false
			}
			return s.isWithinDateRange(info.ModTime())
		},
	})
	if err != nil {
		return fmt.Errorf("创建文件遍历器失败: %w", err)
	}

	err = fileWalker.Walk(ctx, func(ctx context.Context, path string, info os.FileInfo) error {
		if err := s.searchFile(ctx, path, handler); err != nil {
			fmt.Fprintf(os.Stderr, "搜索文件 %s 失败: %v\n", path, err)
			return nil
		}
		return nil
	})

	if err != nil {
		if ctx.Err() != nil && err == ctx.Err() {
			return err
		}
		return fmt.Errorf("遍历日志目录失败: %w", err)
	}

	return nil
}

// searchFile 在单个文件中搜索
func (s *Searcher) searchFile(ctx context.Context, filePath string, handler ResultHandler) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	prevLines := make([]string, 0, s.options.ShowContext)
	var pendings []*pendingContext

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 256*1024)
	scanner.Buffer(buf, 1024*1024)

	lineNum := 0
	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		lineNum++
		line := scanner.Text()

		pendings, err = s.feedPendingContexts(pendings, line, handler)
		if err != nil {
			return err
		}

		if s.matches(line) {
			result := &SearchResult{
				FilePath: filePath,
				LineNum:  lineNum,
				Line:     line,
			}

			if s.options.ShowContext > 0 {
				contextLines := make([]string, len(prevLines))
				copy(contextLines, prevLines)
				contextLines = append(contextLines, line)
				result.Context = contextLines

				if s.options.ShowContext > 0 {
					pendings = append(pendings, &pendingContext{
						result:    result,
						remaining: s.options.ShowContext,
					})
				}
			} else {
				if err := handler(result); err != nil {
					return err
				}
			}
		}

		if s.options.ShowContext > 0 {
			if len(prevLines) == s.options.ShowContext {
				prevLines = prevLines[1:]
			}
			prevLines = append(prevLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if err := flushPendingContexts(pendings, handler); err != nil {
		return err
	}

	return nil
}

func (s *Searcher) feedPendingContexts(pendings []*pendingContext, line string, handler ResultHandler) ([]*pendingContext, error) {
	if len(pendings) == 0 {
		return pendings, nil
	}

	idx := 0
	for _, pending := range pendings {
		if pending.remaining <= 0 {
			if err := handler(pending.result); err != nil {
				return nil, err
			}
			continue
		}
		pending.result.Context = append(pending.result.Context, line)
		pending.remaining--
		if pending.remaining > 0 {
			pendings[idx] = pending
			idx++
			continue
		}
		if err := handler(pending.result); err != nil {
			return nil, err
		}
	}

	return pendings[:idx], nil
}

func flushPendingContexts(pendings []*pendingContext, handler ResultHandler) error {
	for _, pending := range pendings {
		if err := handler(pending.result); err != nil {
			return err
		}
	}
	return nil
}

func containsLowerASCII(line string, needle []byte) bool {
	if len(needle) == 0 {
		return true
	}

	lineBytes := []byte(line)
	if len(lineBytes) < len(needle) {
		return false
	}

	last := len(lineBytes) - len(needle)
	for i := 0; i <= last; i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			if toLowerASCII(lineBytes[i+j]) != needle[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return true
}

func toLowerASCII(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + 32
	}
	return b
}

// matches 检查行是否匹配
func (s *Searcher) matches(line string) bool {
	if s.options.UseRegex {
		return s.regex.MatchString(line)
	}

	if s.options.CaseSensitive {
		return strings.Contains(line, s.options.Keyword)
	}

	if s.useASCIIMatcher {
		return containsLowerASCII(line, s.asciiKeyword)
	}

	return strings.Contains(strings.ToLower(line), s.lowerKeyword)
}

// isWithinDateRange 检查日期是否在范围内
func (s *Searcher) isWithinDateRange(t time.Time) bool {
	if !s.options.StartDate.IsZero() && t.Before(s.options.StartDate) {
		return false
	}
	if !s.options.EndDate.IsZero() && t.After(s.options.EndDate) {
		return false
	}
	return true
}
