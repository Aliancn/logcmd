package reader

import (
	"os"
	"testing"
)

// BenchmarkQuickReadLines 测试快速读取性能
func BenchmarkQuickReadLines(b *testing.B) {
	// 使用真实的测试文件
	testFile := "/tmp/logcmd_perf_test/large.log"

	// 跳过如果文件不存在
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		b.Skip("测试文件不存在，运行 /tmp/test_performance.sh 生成")
	}

	reader, err := NewChunkedReader(testFile, DefaultConfig())
	if err != nil {
		b.Fatalf("创建reader失败: %v", err)
	}
	defer reader.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reader.QuickReadLines(1000)
		if err != nil {
			b.Fatalf("QuickReadLines失败: %v", err)
		}
	}
}

// BenchmarkBuildIndex 测试索引构建性能
func BenchmarkBuildIndex(b *testing.B) {
	testFile := "/tmp/logcmd_perf_test/large.log"

	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		b.Skip("测试文件不存在")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader, err := NewChunkedReader(testFile, DefaultConfig())
		if err != nil {
			b.Fatalf("创建reader失败: %v", err)
		}

		err = reader.BuildIndex()
		if err != nil {
			b.Fatalf("BuildIndex失败: %v", err)
		}

		reader.Close()
	}
}

// BenchmarkReadLines 测试随机行读取性能
func BenchmarkReadLines(b *testing.B) {
	testFile := "/tmp/logcmd_perf_test/large.log"

	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		b.Skip("测试文件不存在")
	}

	reader, err := NewChunkedReader(testFile, DefaultConfig())
	if err != nil {
		b.Fatalf("创建reader失败: %v", err)
	}
	defer reader.Close()

	err = reader.BuildIndex()
	if err != nil {
		b.Fatalf("BuildIndex失败: %v", err)
	}

	totalLines := reader.TotalLines()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 读取100行
		start := i % (totalLines - 100)
		_, err := reader.ReadLines(start, start+99)
		if err != nil {
			b.Fatalf("ReadLines失败: %v", err)
		}
	}
}

// BenchmarkRipgrepSearch 测试ripgrep搜索性能
func BenchmarkRipgrepSearch(b *testing.B) {
	testFile := "/tmp/logcmd_perf_test/large.log"

	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		b.Skip("测试文件不存在")
	}

	if !RipgrepAvailable() {
		b.Skip("ripgrep未安装")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := RipgrepSearch(testFile, "test")
		if err != nil {
			b.Fatalf("RipgrepSearch失败: %v", err)
		}
	}
}

// BenchmarkFallbackSearch 测试fallback搜索性能
func BenchmarkFallbackSearch(b *testing.B) {
	testFile := "/tmp/logcmd_perf_test/large.log"

	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		b.Skip("测试文件不存在")
	}

	reader, err := NewChunkedReader(testFile, DefaultConfig())
	if err != nil {
		b.Fatalf("创建reader失败: %v", err)
	}
	defer reader.Close()

	err = reader.BuildIndex()
	if err != nil {
		b.Fatalf("BuildIndex失败: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := FallbackSearch(reader, "test")
		if err != nil {
			b.Fatalf("FallbackSearch失败: %v", err)
		}
	}
}
