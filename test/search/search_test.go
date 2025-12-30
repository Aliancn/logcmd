package search_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aliancn/logcmd/internal/search"
)

func TestNew(t *testing.T) {
	options := &search.SearchOptions{
		LogDir:  "/tmp",
		Keyword: "test",
	}

	searcher, err := search.New(options)
	if err != nil {
		t.Fatalf("New() 失败: %v", err)
	}

	if searcher == nil {
		t.Fatal("New() 返回了 nil")
	}
}

func TestNewWithRegex(t *testing.T) {
	options := &search.SearchOptions{
		LogDir:   "/tmp",
		Keyword:  "test.*pattern",
		UseRegex: true,
	}

	searcher, err := search.New(options)
	if err != nil {
		t.Fatalf("New() 失败: %v", err)
	}

	if searcher == nil {
		t.Fatal("New() 返回了 nil")
	}
}

func TestNewWithInvalidRegex(t *testing.T) {
	options := &search.SearchOptions{
		LogDir:   "/tmp",
		Keyword:  "[invalid(regex",
		UseRegex: true,
	}

	_, err := search.New(options)
	if err == nil {
		t.Error("New() 应该对无效正则表达式返回错误")
	}
}

func TestSearchEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	options := &search.SearchOptions{
		LogDir:  tmpDir,
		Keyword: "test",
	}

	searcher, err := search.New(options)
	if err != nil {
		t.Fatalf("New() 失败: %v", err)
	}

	results, err := searcher.Search(context.Background())
	if err != nil {
		t.Fatalf("Search() 失败: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("空目录应该返回 0 个结果, got %d", len(results))
	}
}

func TestSearchWithKeyword(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试日志文件
	logContent := `Line 1: some content
Line 2: test keyword here
Line 3: more content
Line 4: another test line
Line 5: final line`

	logPath := filepath.Join(tmpDir, "test.log")
	if err := os.WriteFile(logPath, []byte(logContent), 0644); err != nil {
		t.Fatalf("创建测试日志文件失败: %v", err)
	}

	options := &search.SearchOptions{
		LogDir:  tmpDir,
		Keyword: "test",
	}

	searcher, err := search.New(options)
	if err != nil {
		t.Fatalf("New() 失败: %v", err)
	}

	results, err := searcher.Search(context.Background())
	if err != nil {
		t.Fatalf("Search() 失败: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("应该找到 2 个结果, got %d", len(results))
	}

	// 验证结果
	for _, result := range results {
		if !strings.Contains(result.Line, "test") {
			t.Errorf("结果行应该包含 'test': %s", result.Line)
		}
	}
}

func TestSearchCaseSensitive(t *testing.T) {
	tmpDir := t.TempDir()

	logContent := `Line 1: TEST uppercase
Line 2: test lowercase
Line 3: TeSt mixed case`

	logPath := filepath.Join(tmpDir, "test.log")
	if err := os.WriteFile(logPath, []byte(logContent), 0644); err != nil {
		t.Fatalf("创建测试日志文件失败: %v", err)
	}

	t.Run("大小写敏感", func(t *testing.T) {
		options := &search.SearchOptions{
			LogDir:        tmpDir,
			Keyword:       "test",
			CaseSensitive: true,
		}

		searcher, _ := search.New(options)
		results, _ := searcher.Search(context.Background())

		// 只应该匹配小写的 "test"
		if len(results) != 1 {
			t.Errorf("大小写敏感搜索应该找到 1 个结果, got %d", len(results))
		}
	})

	t.Run("大小写不敏感", func(t *testing.T) {
		options := &search.SearchOptions{
			LogDir:        tmpDir,
			Keyword:       "test",
			CaseSensitive: false,
		}

		searcher, _ := search.New(options)
		results, _ := searcher.Search(context.Background())

		// 应该匹配所有变体
		if len(results) != 3 {
			t.Errorf("大小写不敏感搜索应该找到 3 个结果, got %d", len(results))
		}
	})
}

func TestSearchWithRegex(t *testing.T) {
	tmpDir := t.TempDir()

	logContent := `Line 1: test123
Line 2: test456
Line 3: test
Line 4: testing789`

	logPath := filepath.Join(tmpDir, "test.log")
	if err := os.WriteFile(logPath, []byte(logContent), 0644); err != nil {
		t.Fatalf("创建测试日志文件失败: %v", err)
	}

	options := &search.SearchOptions{
		LogDir:   tmpDir,
		Keyword:  `test\d+`,
		UseRegex: true,
	}

	searcher, err := search.New(options)
	if err != nil {
		t.Fatalf("New() 失败: %v", err)
	}

	results, err := searcher.Search(context.Background())
	if err != nil {
		t.Fatalf("Search() 失败: %v", err)
	}

	// 应该匹配 test123, test456, testing789 (包含数字的)
	if len(results) < 2 {
		t.Errorf("正则表达式搜索应该找到至少 2 个结果, got %d", len(results))
	}
}

func TestSearchWithContext(t *testing.T) {
	tmpDir := t.TempDir()

	logContent := `Line 1
Line 2
Line 3: test keyword
Line 4
Line 5`

	logPath := filepath.Join(tmpDir, "test.log")
	if err := os.WriteFile(logPath, []byte(logContent), 0644); err != nil {
		t.Fatalf("创建测试日志文件失败: %v", err)
	}

	options := &search.SearchOptions{
		LogDir:      tmpDir,
		Keyword:     "test",
		ShowContext: 1, // 显示上下各1行
	}

	searcher, err := search.New(options)
	if err != nil {
		t.Fatalf("New() 失败: %v", err)
	}

	results, err := searcher.Search(context.Background())
	if err != nil {
		t.Fatalf("Search() 失败: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("应该找到 1 个结果, got %d", len(results))
	}

	result := results[0]
	// 上下文应该包含至少1行（匹配行本身）
	if len(result.Context) == 0 {
		t.Error("Context 不应为空")
	}

	// 验证上下文包含匹配的行
	found := false
	for _, line := range result.Context {
		if strings.Contains(line, "test") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Context 应该包含匹配的行")
	}
}

func TestSearchOnlyLogFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建.log文件
	logPath := filepath.Join(tmpDir, "test.log")
	if err := os.WriteFile(logPath, []byte("test keyword"), 0644); err != nil {
		t.Fatalf("创建日志文件失败: %v", err)
	}

	// 创建非.log文件
	txtPath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(txtPath, []byte("test keyword"), 0644); err != nil {
		t.Fatalf("创建文本文件失败: %v", err)
	}

	options := &search.SearchOptions{
		LogDir:  tmpDir,
		Keyword: "test",
	}

	searcher, err := search.New(options)
	if err != nil {
		t.Fatalf("New() 失败: %v", err)
	}

	results, err := searcher.Search(context.Background())
	if err != nil {
		t.Fatalf("Search() 失败: %v", err)
	}

	// 应该只搜索.log文件
	if len(results) != 1 {
		t.Errorf("应该只搜索.log文件, 找到 %d 个结果", len(results))
	}

	if !strings.HasSuffix(results[0].FilePath, ".log") {
		t.Error("结果应该来自.log文件")
	}
}

func TestSearchWithDateRange(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试文件
	logPath := filepath.Join(tmpDir, "test.log")
	if err := os.WriteFile(logPath, []byte("test content"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 设置文件的修改时间为过去某个时间
	pastTime := time.Now().AddDate(0, 0, -10)
	if err := os.Chtimes(logPath, pastTime, pastTime); err != nil {
		t.Fatalf("设置文件时间失败: %v", err)
	}

	t.Run("超出日期范围", func(t *testing.T) {
		options := &search.SearchOptions{
			LogDir:    tmpDir,
			Keyword:   "test",
			StartDate: time.Now().AddDate(0, 0, -5), // 从5天前开始
		}

		searcher, _ := search.New(options)
		results, _ := searcher.Search(context.Background())

		// 文件是10天前的，不应该被搜索
		if len(results) != 0 {
			t.Errorf("超出日期范围的文件不应被搜索, got %d 个结果", len(results))
		}
	})

	t.Run("在日期范围内", func(t *testing.T) {
		options := &search.SearchOptions{
			LogDir:    tmpDir,
			Keyword:   "test",
			StartDate: time.Now().AddDate(0, 0, -15), // 从15天前开始
		}

		searcher, _ := search.New(options)
		results, _ := searcher.Search(context.Background())

		// 文件是10天前的，应该被搜索到
		if len(results) != 1 {
			t.Errorf("日期范围内的文件应被搜索, got %d 个结果", len(results))
		}
	})
}

func TestSearchResultFields(t *testing.T) {
	tmpDir := t.TempDir()

	logContent := "Line 1: test keyword"
	logPath := filepath.Join(tmpDir, "test.log")
	if err := os.WriteFile(logPath, []byte(logContent), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	options := &search.SearchOptions{
		LogDir:  tmpDir,
		Keyword: "test",
	}

	searcher, _ := search.New(options)
	results, _ := searcher.Search(context.Background())

	if len(results) != 1 {
		t.Fatalf("应该找到 1 个结果")
	}

	result := results[0]

	// 验证结果字段
	if result.FilePath == "" {
		t.Error("FilePath 不应为空")
	}

	if result.LineNum != 1 {
		t.Errorf("LineNum = %d, want 1", result.LineNum)
	}

	if result.Line == "" {
		t.Error("Line 不应为空")
	}

	if !strings.Contains(result.Line, "test") {
		t.Error("Line 应该包含搜索关键词")
	}
}

func TestPrintResults(t *testing.T) {
	results := []*search.SearchResult{
		{
			FilePath: "/path/to/test.log",
			LineNum:  5,
			Line:     "test line",
		},
	}

	// PrintResults 应该不会 panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PrintResults() panic: %v", r)
		}
	}()

	search.PrintResults(results)
}

func TestPrintResultsEmpty(t *testing.T) {
	var results []*search.SearchResult

	// PrintResults 应该能处理空结果
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PrintResults() panic on empty results: %v", r)
		}
	}()

	search.PrintResults(results)
}

func TestSearchNonExistentDirectory(t *testing.T) {
	options := &search.SearchOptions{
		LogDir:  "/nonexistent/directory/12345",
		Keyword: "test",
	}

	searcher, err := search.New(options)
	if err != nil {
		t.Fatalf("New() 失败: %v", err)
	}

	_, err = searcher.Search(context.Background())
	if err == nil {
		t.Error("Search() 应该对不存在的目录返回错误")
	}
}
