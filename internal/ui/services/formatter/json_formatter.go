package formatter

import (
	"encoding/json"
	"strings"

	"github.com/tidwall/pretty"
)

// JSONFormatter JSON格式化器
type JSONFormatter struct {
	colorful bool
	indent   int
}

// NewJSONFormatter 创建新的JSON格式化器
func NewJSONFormatter(colorful bool) *JSONFormatter {
	return &JSONFormatter{
		colorful: colorful,
		indent:   2,
	}
}

// SetColorful 设置是否使用彩色输出
func (f *JSONFormatter) SetColorful(colorful bool) {
	f.colorful = colorful
}

// SetIndent 设置缩进空格数
func (f *JSONFormatter) SetIndent(indent int) {
	f.indent = indent
}

// Format 格式化JSON字符串
func (f *JSONFormatter) Format(jsonStr string) (string, error) {
	// 验证JSON有效性
	if !json.Valid([]byte(jsonStr)) {
		return jsonStr, nil // 如果不是有效JSON，返回原文
	}

	// 使用pretty进行格式化
	var result []byte
	if f.colorful {
		// 彩色输出
		result = pretty.Color([]byte(jsonStr), nil)
	} else {
		// 纯文本美化
		result = pretty.Pretty([]byte(jsonStr))
	}

	// 应用缩进
	if f.indent != 2 {
		result = pretty.PrettyOptions(result, &pretty.Options{
			Width:  80,
			Prefix: "",
			Indent: strings.Repeat(" ", f.indent),
		})
	}

	return string(result), nil
}

// ExtractJSON 从日志行中提取JSON部分
// 返回值: prefix(JSON前的文本), json(JSON内容), suffix(JSON后的文本), found(是否找到JSON)
func ExtractJSON(line string) (prefix, jsonPart, suffix string, found bool) {
	// 查找JSON对象 {...}
	startObj := strings.Index(line, "{")
	if startObj != -1 {
		if endObj := findMatchingBrace(line, startObj); endObj != -1 {
			prefix = line[:startObj]
			jsonPart = line[startObj : endObj+1]
			if endObj+1 < len(line) {
				suffix = line[endObj+1:]
			}
			// 验证是否为有效JSON
			if json.Valid([]byte(jsonPart)) {
				return prefix, jsonPart, suffix, true
			}
		}
	}

	// 查找JSON数组 [...]
	startArr := strings.Index(line, "[")
	if startArr != -1 {
		if endArr := findMatchingBracket(line, startArr); endArr != -1 {
			prefix = line[:startArr]
			jsonPart = line[startArr : endArr+1]
			if endArr+1 < len(line) {
				suffix = line[endArr+1:]
			}
			// 验证是否为有效JSON
			if json.Valid([]byte(jsonPart)) {
				return prefix, jsonPart, suffix, true
			}
		}
	}

	return "", "", "", false
}

// findMatchingBrace 查找匹配的}
func findMatchingBrace(s string, start int) int {
	depth := 0
	inString := false
	escape := false

	for i := start; i < len(s); i++ {
		c := s[i]

		if escape {
			escape = false
			continue
		}

		if c == '\\' {
			escape = true
			continue
		}

		if c == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		switch c {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// findMatchingBracket 查找匹配的]
func findMatchingBracket(s string, start int) int {
	depth := 0
	inString := false
	escape := false

	for i := start; i < len(s); i++ {
		c := s[i]

		if escape {
			escape = false
			continue
		}

		if c == '\\' {
			escape = true
			continue
		}

		if c == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		switch c {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// FormatLogLine 格式化包含JSON的日志行
func (f *JSONFormatter) FormatLogLine(line string) string {
	prefix, jsonPart, suffix, found := ExtractJSON(line)
	if !found {
		return line
	}

	// 格式化JSON部分
	formatted, err := f.Format(jsonPart)
	if err != nil {
		return line
	}

	// 重组行
	// 对于多行JSON，保留prefix在第一行，suffix在最后一行
	lines := strings.Split(formatted, "\n")
	if len(lines) == 1 {
		return prefix + formatted + suffix
	}

	// 多行JSON
	result := prefix + lines[0] + "\n"
	for i := 1; i < len(lines)-1; i++ {
		result += lines[i] + "\n"
	}
	result += lines[len(lines)-1] + suffix

	return result
}

// IsJSON 检查字符串是否为有效JSON
func IsJSON(s string) bool {
	trimmed := strings.TrimSpace(s)
	if len(trimmed) == 0 {
		return false
	}

	// 快速检查首尾字符
	first := trimmed[0]
	last := trimmed[len(trimmed)-1]

	if (first == '{' && last == '}') || (first == '[' && last == ']') {
		return json.Valid([]byte(trimmed))
	}

	return false
}
