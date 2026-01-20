package reader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChunkedReader_BuildIndex(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.log")

	// 1. Test with regular newlines
	content := "line1\nline2\nline3\nline4\nline5\n"
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	config := Config{IndexInterval: 2} // Index every 2 lines
	reader, err := NewChunkedReader(filePath, config)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}
	defer reader.Close()

	err = reader.BuildIndex()
	if err != nil {
		t.Fatalf("Failed to build index: %v", err)
	}

	if reader.TotalLines() != 5 {
		t.Errorf("Expected 5 lines, got %d", reader.TotalLines())
	}

	// Read specific lines to verify index accuracy
	lines, err := reader.ReadLines(0, 4)
	if err != nil {
		t.Fatalf("Failed to read lines: %v", err)
	}
	if len(lines) != 5 {
		t.Errorf("Expected 5 lines, got %d", len(lines))
	}
	if lines[0] != "line1" {
		t.Errorf("Expected line1, got %s", lines[0])
	}
	if lines[4] != "line5" {
		t.Errorf("Expected line5, got %s", lines[4])
	}

	// 2. Test without trailing newline
	contentNoNL := "line1\nline2\nline3"
	err = os.WriteFile(filePath, []byte(contentNoNL), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	reader2, err := NewChunkedReader(filePath, config)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}
	defer reader2.Close()

	err = reader2.BuildIndex()
	if err != nil {
		t.Fatalf("Failed to build index: %v", err)
	}

	if reader2.TotalLines() != 3 {
		t.Errorf("Expected 3 lines, got %d", reader2.TotalLines())
	}

	lines2, err := reader2.ReadLines(0, 2)
	if err != nil {
		t.Fatalf("Failed to read lines: %v", err)
	}
	if len(lines2) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines2))
	}
	if lines2[2] != "line3" {
		t.Errorf("Expected line3, got %s", lines2[2])
	}
}

func TestChunkedReader_LargeFile(t *testing.T) {
	// Create a larger file (simulate larger content, though actual 100MB is too slow for unit test)
	// We'll create enough to trigger multiple buffer reads (buffer is 64KB)
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "large.log")

	line := "This is a test line with some content to fill up space.\n"
	lineLen := len(line)
	// 64KB buffer. Let's write 128KB+
	numLines := (128 * 1024) / lineLen + 100
	
f, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	for i := 0; i < numLines; i++ {
		f.WriteString(line)
	}
	f.Close()

	reader, err := NewChunkedReader(filePath, Config{IndexInterval: 100})
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}
	defer reader.Close()

	err = reader.BuildIndex()
	if err != nil {
		t.Fatalf("Failed to build index: %v", err)
	}

	if reader.TotalLines() != numLines {
		t.Errorf("Expected %d lines, got %d", numLines, reader.TotalLines())
	}

	// Read lines from the middle (crossing buffer boundaries)
	mid := numLines / 2
	lines, err := reader.ReadLines(mid, mid)
	if err != nil {
		t.Fatalf("Failed to read line %d: %v", mid, err)
	}
	if len(lines) != 1 {
		t.Fatalf("Expected 1 line, got %d", len(lines))
	}
	if !strings.HasPrefix(lines[0], "This is a test line") {
		t.Errorf("Content mismatch at line %d: %s", mid, lines[0])
	}
}
