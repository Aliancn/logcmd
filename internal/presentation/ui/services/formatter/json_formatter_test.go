package formatter

import (
	"testing"
)

// TestExtractJSON_FastFail 测试快速失败优化
func TestExtractJSON_FastFail(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected bool
	}{
		{
			name:     "纯文本日志（无JSON字符）",
			line:     "[STDOUT] This is a plain text log line",
			expected: false,
		},
		{
			name:     "包含JSON对象",
			line:     `[INFO] User logged in {"user": "alice", "time": 123}`,
			expected: true,
		},
		{
			name:     "包含JSON数组",
			line:     `Values: [1, 2, 3, 4, 5]`,
			expected: true,
		},
		{
			name:     "空行",
			line:     "",
			expected: false,
		},
		{
			name:     "只有方括号（非JSON）",
			line:     "[STDOUT] Some text [not json]",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, found := ExtractJSON(tt.line)
			if found != tt.expected {
				t.Errorf("ExtractJSON(%q) = %v, want %v", tt.line, found, tt.expected)
			}
		})
	}
}

// BenchmarkExtractJSON_PlainText 基准测试：纯文本日志（快速失败路径）
func BenchmarkExtractJSON_PlainText(b *testing.B) {
	line := "[STDOUT] This is a plain text log line without any JSON content"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExtractJSON(line)
	}
}

// BenchmarkExtractJSON_WithJSON 基准测试：包含JSON的日志
func BenchmarkExtractJSON_WithJSON(b *testing.B) {
	line := `[INFO] Request completed {"method": "GET", "path": "/api/users", "status": 200, "duration": 45}`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExtractJSON(line)
	}
}

// BenchmarkExtractJSON_FakeBracket 基准测试：包含非JSON方括号
func BenchmarkExtractJSON_FakeBracket(b *testing.B) {
	line := "[STDOUT] Process completed [exit code: 0]"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExtractJSON(line)
	}
}
