package highlighter

import (
	"bytes"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// LogFormat 定义日志格式类型
type LogFormat string

const (
	// FormatAuto 自动检测格式
	FormatAuto LogFormat = "auto"
	// FormatPlain 纯文本（无高亮）
	FormatPlain LogFormat = "plain"
	// FormatJSON JSON格式
	FormatJSON LogFormat = "json"
	// FormatLogfmt Logfmt格式
	FormatLogfmt LogFormat = "logfmt"
)

// Highlighter 定义语法高亮接口
type Highlighter interface {
	// HighlightLine 对单行进行高亮处理
	HighlightLine(line string, lineNum int) string
	// SetFormat 设置日志格式
	SetFormat(format LogFormat)
	// SetTheme 设置主题
	SetTheme(theme string)
	// GetFormat 获取当前格式
	GetFormat() LogFormat
}

// ChromaHighlighter 基于Chroma的语法高亮实现
type ChromaHighlighter struct {
	format    LogFormat
	themeName string
	style     *chroma.Style
	formatter chroma.Formatter
}

// NewHighlighter 创建新的语法高亮器
func NewHighlighter() *ChromaHighlighter {
	h := &ChromaHighlighter{
		format:    FormatAuto,
		themeName: "monokai",
	}
	h.updateFormatter()
	return h
}

// updateFormatter 更新formatter和style
func (h *ChromaHighlighter) updateFormatter() {
	h.style = styles.Get(h.themeName)
	if h.style == nil {
		h.style = styles.Fallback
	}
	h.formatter = formatters.Get("terminal256")
	if h.formatter == nil {
		h.formatter = formatters.Fallback
	}
}

// SetFormat 设置日志格式
func (h *ChromaHighlighter) SetFormat(format LogFormat) {
	h.format = format
}

// SetTheme 设置主题
func (h *ChromaHighlighter) SetTheme(theme string) {
	h.themeName = theme
	h.updateFormatter()
}

// GetFormat 获取当前格式
func (h *ChromaHighlighter) GetFormat() LogFormat {
	return h.format
}

// HighlightLine 对单行进行高亮处理
func (h *ChromaHighlighter) HighlightLine(line string, lineNum int) string {
	// 如果是纯文本格式，直接返回
	if h.format == FormatPlain {
		return line
	}

	// 检测并应用格式
	format := h.format
	if format == FormatAuto {
		format = h.detectFormat(line)
	}

	// 根据格式选择lexer
	var lexer chroma.Lexer
	switch format {
	case FormatJSON:
		lexer = lexers.Get("json")
	case FormatLogfmt:
		// Logfmt使用ini lexer近似
		lexer = lexers.Get("ini")
	default:
		// 尝试分析日志格式
		lexer = h.getLogLexer(line)
	}

	// 如果无法确定lexer，返回原文
	if lexer == nil {
		return line
	}

	// 使用Chroma进行高亮
	iterator, err := lexer.Tokenise(nil, line)
	if err != nil {
		return line
	}

	var buf bytes.Buffer
	err = h.formatter.Format(&buf, h.style, iterator)
	if err != nil {
		return line
	}

	return buf.String()
}

// detectFormat 自动检测日志格式
func (h *ChromaHighlighter) detectFormat(line string) LogFormat {
	trimmed := strings.TrimSpace(line)

	// 检测JSON
	if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
		return FormatJSON
	}
	if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
		return FormatJSON
	}

	// 检测Logfmt (key=value格式)
	if strings.Contains(trimmed, "=") && !strings.Contains(trimmed, "{") {
		parts := strings.Fields(trimmed)
		kvCount := 0
		for _, part := range parts {
			if strings.Contains(part, "=") {
				kvCount++
			}
		}
		// 如果大部分字段都是key=value格式
		if kvCount >= len(parts)/2 {
			return FormatLogfmt
		}
	}

	// 默认返回auto（使用通用日志lexer）
	return FormatAuto
}

// getLogLexer 获取通用日志lexer
func (h *ChromaHighlighter) getLogLexer(line string) chroma.Lexer {
	// 尝试检测常见的日志模式
	if strings.Contains(line, "ERROR") || strings.Contains(line, "WARN") ||
	   strings.Contains(line, "INFO") || strings.Contains(line, "DEBUG") {
		// 使用accesslog lexer提供基础日志高亮
		lexer := lexers.Get("accesslog")
		if lexer != nil {
			return lexer
		}
	}

	// 返回nil让调用者处理
	return nil
}

// SimpleHighlight 提供简单的高亮方案（不依赖Chroma，用于降级）
func SimpleHighlight(line string) string {
	// 简单的关键字着色
	result := line

	// 错误级别 - 红色
	result = strings.ReplaceAll(result, "ERROR", "\033[31mERROR\033[0m")
	result = strings.ReplaceAll(result, "FATAL", "\033[31;1mFATAL\033[0m")
	result = strings.ReplaceAll(result, "error", "\033[31merror\033[0m")

	// 警告级别 - 黄色
	result = strings.ReplaceAll(result, "WARN", "\033[33mWARN\033[0m")
	result = strings.ReplaceAll(result, "WARNING", "\033[33mWARNING\033[0m")
	result = strings.ReplaceAll(result, "warn", "\033[33mwarn\033[0m")

	// 信息级别 - 绿色
	result = strings.ReplaceAll(result, "INFO", "\033[32mINFO\033[0m")
	result = strings.ReplaceAll(result, "info", "\033[32minfo\033[0m")

	// 调试级别 - 蓝色
	result = strings.ReplaceAll(result, "DEBUG", "\033[34mDEBUG\033[0m")
	result = strings.ReplaceAll(result, "debug", "\033[34mdebug\033[0m")
	result = strings.ReplaceAll(result, "TRACE", "\033[36mTRACE\033[0m")

	return result
}
