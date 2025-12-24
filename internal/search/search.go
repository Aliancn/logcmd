package search

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// SearchOptions 搜索选项
type SearchOptions struct {
	LogDir      string    // 日志目录
	Keyword     string    // 搜索关键词
	UseRegex    bool      // 使用正则表达式
	StartDate   time.Time // 开始日期
	EndDate     time.Time // 结束日期
	CaseSensitive bool    // 区分大小写
	ShowContext int       // 显示上下文行数
}

// SearchResult 搜索结果
type SearchResult struct {
	FilePath  string   // 文件路径
	LineNum   int      // 行号
	Line      string   // 匹配的行
	Context   []string // 上下文行
}

// Searcher 日志搜索器
type Searcher struct {
	options *SearchOptions
	regex   *regexp.Regexp
}

// New 创建搜索器
func New(options *SearchOptions) (*Searcher, error) {
	s := &Searcher{
		options: options,
	}

	// 如果使用正则表达式，编译它
	if options.UseRegex {
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

	return s, nil
}

// Search 执行搜索
func (s *Searcher) Search() ([]*SearchResult, error) {
	var results []*SearchResult

	// 遍历日志目录
	err := filepath.Walk(s.options.LogDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
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
		fileResults, err := s.searchFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "搜索文件 %s 失败: %v\n", path, err)
			return nil // 继续搜索其他文件
		}

		results = append(results, fileResults...)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("遍历日志目录失败: %w", err)
	}

	return results, nil
}

// searchFile 在单个文件中搜索
func (s *Searcher) searchFile(filePath string) ([]*SearchResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var results []*SearchResult
	var lines []string // 保存所有行用于上下文

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 256*1024)
	scanner.Buffer(buf, 1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		lines = append(lines, line)

		if s.matches(line) {
			result := &SearchResult{
				FilePath: filePath,
				LineNum:  lineNum,
				Line:     line,
			}

			// 添加上下文
			if s.options.ShowContext > 0 {
				result.Context = s.getContext(lines, lineNum-1)
			}

			results = append(results, result)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return results, nil
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

// getContext 获取上下文行
func (s *Searcher) getContext(lines []string, currentIdx int) []string {
	start := currentIdx - s.options.ShowContext
	if start < 0 {
		start = 0
	}

	end := currentIdx + s.options.ShowContext + 1
	if end > len(lines) {
		end = len(lines)
	}

	return lines[start:end]
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
