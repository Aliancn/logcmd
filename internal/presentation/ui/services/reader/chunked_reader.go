package reader

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
)

// ChunkedReader 分块读取器，用于高效处理大文件
type ChunkedReader struct {
	filePath         string
	file             *os.File
	lineIndex        []int64 // 行偏移索引（每N行记录一次）
	totalLines       int
	indexInterval    int // 索引间隔（默认100行）
	fileSize         int64
	mu               sync.RWMutex
	indexed          bool
	indexingProgress float64 // 索引进度 0.0-1.0
}

// Config 读取器配置
type Config struct {
	IndexInterval int // 索引间隔，默认100
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		IndexInterval: 100,
	}
}

// NewChunkedReader 创建新的分块读取器
func NewChunkedReader(filePath string, config Config) (*ChunkedReader, error) {
	if config.IndexInterval <= 0 {
		config.IndexInterval = 100
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}

	fileInfo, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}

	return &ChunkedReader{
		filePath:      filePath,
		file:          file,
		lineIndex:     make([]int64, 0, 1000),
		indexInterval: config.IndexInterval,
		fileSize:      fileInfo.Size(),
		indexed:       false,
	}, nil
}

// BuildIndex 构建行索引
// 这是一个耗时操作，应该在后台异步执行
func (r *ChunkedReader) BuildIndex() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.indexed {
		return nil // 已经建立索引
	}

	// 重置文件指针到开头
	_, err := r.file.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("重置文件指针失败: %w", err)
	}

	// 使用大缓冲区进行扫描，避免scanner的行长度限制和内存分配开销
	const bufSize = 64 * 1024 // 64KB buffer
	buf := make([]byte, bufSize)

	lineNum := 0
	offset := int64(0)

	// 记录第一行的偏移（总是0）
	r.lineIndex = append(r.lineIndex, 0)

	for {
		n, err := r.file.Read(buf)
		if n > 0 {
			data := buf[:n]
			pos := 0
			for {
				i := bytes.IndexByte(data[pos:], '\n')
				if i == -1 {
					break
				}

				// 找到新的一行
				// 换行符位置: offset + pos + i
				// 下一行起始位置: offset + pos + i + 1
				lineNum++

				// 如果是索引点，记录下一行的起始位置
				if lineNum%r.indexInterval == 0 {
					nextLineOffset := offset + int64(pos+i+1)
					r.lineIndex = append(r.lineIndex, nextLineOffset)
				}

				pos += i + 1
			}
			offset += int64(n)

			// 更新索引进度
			r.indexingProgress = float64(offset) / float64(r.fileSize)
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("扫描文件失败: %w", err)
		}
	}

	// 处理最后一行（如果不以换行符结尾）
	// offset此时是文件总长度
	// 如果文件不为空且最后一个字符不是换行符，则需要增加一行计数
	if offset > 0 {
		// 需要检查最后一个字节是否是换行符
		// 为了简单起见，我们回退一个字节读取
		_, err := r.file.Seek(-1, io.SeekEnd)
		if err == nil {
			lastByte := make([]byte, 1)
			r.file.Read(lastByte)
			if lastByte[0] != '\n' {
				lineNum++
			}
		}
	}

	r.totalLines = lineNum
	r.indexed = true
	r.indexingProgress = 1.0

	return nil
}

// TotalLines 返回文件总行数
func (r *ChunkedReader) TotalLines() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.totalLines
}

// FileSize 返回文件大小
func (r *ChunkedReader) FileSize() int64 {
	return r.fileSize
}

// IsIndexed 返回是否已建立索引
func (r *ChunkedReader) IsIndexed() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.indexed
}

// ReadLines 读取指定范围的行 (start和end都是从0开始的索引)
func (r *ChunkedReader) ReadLines(start, end int) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.indexed {
		return nil, fmt.Errorf("索引未建立，请先调用BuildIndex()")
	}

	if start < 0 {
		start = 0
	}
	if end >= r.totalLines {
		end = r.totalLines - 1
	}
	if start > end {
		return []string{}, nil
	}

	// 计算起始偏移
	indexPos := start / r.indexInterval
	if indexPos >= len(r.lineIndex) {
		return nil, fmt.Errorf("索引位置超出范围")
	}

	startOffset := r.lineIndex[indexPos]

	// Seek到索引位置
	_, err := r.file.Seek(startOffset, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("定位文件失败: %w", err)
	}

	scanner := bufio.NewScanner(r.file)
	// 设置较大的缓冲区
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	// 跳过索引点到实际起始行之间的行
	skipLines := start - (indexPos * r.indexInterval)
	for i := 0; i < skipLines && scanner.Scan(); i++ {
		// 只是跳过，不读取内容
	}

	// 读取目标行
	lines := make([]string, 0, end-start+1)
	currentLine := start

	for currentLine <= end && scanner.Scan() {
		lines = append(lines, scanner.Text())
		currentLine++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取行失败: %w", err)
	}

	return lines, nil
}

// ReadLine 读取单行
func (r *ChunkedReader) ReadLine(lineNum int) (string, error) {
	lines, err := r.ReadLines(lineNum, lineNum)
	if err != nil {
		return "", err
	}
	if len(lines) == 0 {
		return "", fmt.Errorf("行号超出范围: %d", lineNum)
	}
	return lines[0], nil
}

// Close 关闭文件
func (r *ChunkedReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.file != nil {
		err := r.file.Close()
		r.file = nil
		return err
	}
	return nil
}

// Reopen 重新打开文件（用于follow模式）
func (r *ChunkedReader) Reopen() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.file != nil {
		r.file.Close()
	}

	file, err := os.Open(r.filePath)
	if err != nil {
		return fmt.Errorf("重新打开文件失败: %w", err)
	}

	r.file = file
	r.indexed = false
	r.lineIndex = make([]int64, 0, 1000)

	return nil
}

// GetIndexProgress 获取索引构建进度（0.0-1.0）
// 注意：这需要在BuildIndex()中配合实现进度回调
func (r *ChunkedReader) GetIndexProgress() float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.indexed {
		return 1.0
	}

	return r.indexingProgress
}

// ShouldUseChunkedRead 判断是否应该使用分块读取
// 基于文件大小做判断
func ShouldUseChunkedRead(fileSize int64, threshold int64) bool {
	if threshold <= 0 {
		threshold = 10 * 1024 * 1024 // 默认10MB
	}
	return fileSize > threshold
}

// QuickReadLines 快速读取文件头部行（不依赖索引）
// 用于在索引构建前提供快速首屏显示
// maxLines: 最多读取的行数
func (r *ChunkedReader) QuickReadLines(maxLines int) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 重置到文件开头
	_, err := r.file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("定位文件失败: %w", err)
	}

	scanner := bufio.NewScanner(r.file)
	// 设置较大的缓冲区
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	lines := make([]string, 0, maxLines)
	count := 0

	for scanner.Scan() && count < maxLines {
		lines = append(lines, scanner.Text())
		count++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取行失败: %w", err)
	}

	return lines, nil
}
