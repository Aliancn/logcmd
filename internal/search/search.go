package search

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
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
	options *SearchOptions
	regex   *regexp.Regexp
}

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
	}

	return s, nil
}

// Search 执行搜索
func (s *Searcher) Search(ctx context.Context) ([]*SearchResult, error) {
	var results []*SearchResult

	// 遍历日志目录
	err := filepath.Walk(s.options.LogDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 只处理.log文件
		if !strings.HasSuffix(path, ".log") {
			return nil
		}

		// 检查日期范围
		if !s.isWithinDateRange(info.ModTime()) {
			return nil
		}

		// 在文件中搜索
		fileResults, err := s.searchFile(ctx, path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "搜索文件 %s 失败: %v\n", path, err)
			return nil // 继续搜索其他文件
		}

		results = append(results, fileResults...)
		return nil
	})

	if err != nil {
		if ctx.Err() != nil && err == ctx.Err() {
			return nil, err
		}
		return nil, fmt.Errorf("遍历日志目录失败: %w", err)
	}

	return results, nil
}

// searchFile 在单个文件中搜索
func (s *Searcher) searchFile(ctx context.Context, filePath string) ([]*SearchResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var results []*SearchResult
	prevLines := make([]string, 0, s.options.ShowContext)
	var pendings []*pendingContext

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 256*1024)
	scanner.Buffer(buf, 1024*1024)

	lineNum := 0
	for scanner.Scan() {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		lineNum++
		line := scanner.Text()

		pendings = s.feedPendingContexts(pendings, line)

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
			}

			results = append(results, result)
		}

		if s.options.ShowContext > 0 {
			if len(prevLines) == s.options.ShowContext {
				prevLines = prevLines[1:]
			}
			prevLines = append(prevLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func (s *Searcher) feedPendingContexts(pendings []*pendingContext, line string) []*pendingContext {
	if len(pendings) == 0 {
		return pendings
	}

	idx := 0
	for _, pending := range pendings {
		if pending.remaining <= 0 {
			continue
		}
		pending.result.Context = append(pending.result.Context, line)
		pending.remaining--
		if pending.remaining > 0 {
			pendings[idx] = pending
			idx++
		}
	}

	return pendings[:idx]
}

// matches 检查行是否匹配
func (s *Searcher) matches(line string) bool {
	if s.options.UseRegex {
		return s.regex.MatchString(line)
	}

	// 普通字符串匹配
	searchLine := line
	keyword := s.options.Keyword

	if !s.options.CaseSensitive {
		searchLine = strings.ToLower(line)
		keyword = strings.ToLower(keyword)
	}

	return strings.Contains(searchLine, keyword)
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

// PrintResults 打印搜索结果
func PrintResults(results []*SearchResult) {
	if len(results) == 0 {
		fmt.Println("未找到匹配的日志")
		return
	}

	fmt.Printf("找到 %d 条匹配记录:\n\n", len(results))

	for _, result := range results {
		fmt.Printf("文件: %s:%d\n", result.FilePath, result.LineNum)
		if len(result.Context) > 0 {
			fmt.Println("上下文:")
			for _, line := range result.Context {
				fmt.Printf("  %s\n", line)
			}
		} else {
			fmt.Printf("  %s\n", result.Line)
		}
		fmt.Println()
	}
}
