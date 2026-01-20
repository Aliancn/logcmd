package reader

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// SearchResult 搜索结果
type SearchResult struct {
	LineNumber int    // 行号（0-based）
	LineText   string // 行内容
}

// RipgrepSearch 使用ripgrep进行快速搜索
// 返回所有匹配行的行号（0-based）
func RipgrepSearch(filePath, query string) ([]int, error) {
	// 检查ripgrep是否可用
	_, err := exec.LookPath("rg")
	if err != nil {
		return nil, fmt.Errorf("ripgrep未安装: %w", err)
	}

	// 使用ripgrep搜索
	// -n: 显示行号
	// --line-number: 显示行号
	// -i: 忽略大小写
	// --no-heading: 不显示文件名
	// --color=never: 不使用颜色
	cmd := exec.Command("rg", "-n", "-i", "--no-heading", "--color=never", query, filePath)

	var out bytes.Buffer
	cmd.Stdout = &out

	err = cmd.Run()
	if err != nil {
		// Exit code 1 表示没有匹配项，不算错误
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return []int{}, nil
		}
		return nil, fmt.Errorf("ripgrep执行失败: %w", err)
	}

	// 解析输出
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	matches := make([]int, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		// ripgrep输出格式: "行号:内容"
		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue
		}

		lineNum, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		// 转换为0-based索引
		matches = append(matches, lineNum-1)
	}

	return matches, nil
}

// RipgrepAvailable 检查ripgrep是否可用
func RipgrepAvailable() bool {
	_, err := exec.LookPath("rg")
	return err == nil
}

// FallbackSearch 回退搜索方法（扫描文件）
// 用于ripgrep不可用时的备选方案
func FallbackSearch(reader *ChunkedReader, query string) ([]int, error) {
	if !reader.IsIndexed() {
		return nil, fmt.Errorf("索引未建立")
	}

	lowerQuery := strings.ToLower(query)
	matches := make([]int, 0)

	totalLines := reader.TotalLines()
	batchSize := 1000 // 每次读取1000行

	for start := 0; start < totalLines; start += batchSize {
		end := start + batchSize - 1
		if end >= totalLines {
			end = totalLines - 1
		}

		lines, err := reader.ReadLines(start, end)
		if err != nil {
			return nil, fmt.Errorf("读取行失败: %w", err)
		}

		for i, line := range lines {
			if strings.Contains(strings.ToLower(line), lowerQuery) {
				matches = append(matches, start+i)
			}
		}
	}

	return matches, nil
}
