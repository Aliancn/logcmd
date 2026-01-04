package walker_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aliancn/logcmd/internal/walker"
)

func TestNewWalkerValidatesRoot(t *testing.T) {
	if _, err := walker.New(walker.Options{}); err == nil {
		t.Fatal("root 为空时应该返回错误")
	}

	dir := t.TempDir()
	w, err := walker.New(walker.Options{Root: dir})
	if err != nil {
		t.Fatalf("New() 失败: %v", err)
	}

	if w == nil {
		t.Fatal("walker 不应为 nil")
	}
}

func TestWalkerProcessesAllFiles(t *testing.T) {
	root := t.TempDir()
	createFiles(t, root, "a.log", "b.log", "sub/c.log")

	w, err := walker.New(walker.Options{Root: root, Workers: 2})
	if err != nil {
		t.Fatalf("New() 失败: %v", err)
	}

	processed := make(map[string]struct{})
	var mu sync.Mutex
	err = w.Walk(context.Background(), func(ctx context.Context, path string, info os.FileInfo) error {
		mu.Lock()
		processed[filepath.Base(path)] = struct{}{}
		mu.Unlock()
		return nil
	})
	if err != nil {
		t.Fatalf("Walk() 失败: %v", err)
	}

	if len(processed) != 3 {
		t.Fatalf("处理的文件数量不正确: %d", len(processed))
	}
}

func TestWalkerFilterIsApplied(t *testing.T) {
	root := t.TempDir()
	createFiles(t, root, "keep.log", "drop.txt", "nested/keep2.log")

	w, err := walker.New(walker.Options{
		Root: root,
		FileFilter: func(path string, info os.FileInfo) bool {
			return filepath.Ext(path) == ".log"
		},
	})
	if err != nil {
		t.Fatalf("New() 失败: %v", err)
	}

	var count int32
	err = w.Walk(context.Background(), func(ctx context.Context, path string, info os.FileInfo) error {
		atomic.AddInt32(&count, 1)
		return nil
	})
	if err != nil {
		t.Fatalf("Walk() 失败: %v", err)
	}

	if count != 2 {
		t.Fatalf("过滤器未生效: count = %d", count)
	}
}

func TestWalkerStopsOnProcessorError(t *testing.T) {
	root := t.TempDir()
	createFiles(t, root, "fail.log", "other.log")

	w, err := walker.New(walker.Options{Root: root})
	if err != nil {
		t.Fatalf("New() 失败: %v", err)
	}

	expectedErr := errors.New("processor error")
	err = w.Walk(context.Background(), func(ctx context.Context, path string, info os.FileInfo) error {
		return expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("Walk() 返回错误不正确: %v", err)
	}
}

func TestWalkerHonorsContextCancellation(t *testing.T) {
	root := t.TempDir()
	createFiles(t, root, "one.log", "two.log")

	w, err := walker.New(walker.Options{Root: root, Workers: 4})
	if err != nil {
		t.Fatalf("New() 失败: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	var once sync.Once
	err = w.Walk(ctx, func(ctx context.Context, path string, info os.FileInfo) error {
		once.Do(func() {
			cancel()
		})
		time.Sleep(5 * time.Millisecond)
		return nil
	})

	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("Walk() 应返回 context.Canceled, got %v", err)
	}
}

func createFiles(t *testing.T, root string, names ...string) {
	t.Helper()
	for _, name := range names {
		fullPath := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("创建目录失败: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(name), 0644); err != nil {
			t.Fatalf("创建文件失败: %v", err)
		}
	}
}
