package walker

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// FileProcessor 定义单个文件的处理逻辑
type FileProcessor func(ctx context.Context, path string, info os.FileInfo) error

// FileFilter 用于过滤需要处理的文件
type FileFilter func(path string, info os.FileInfo) bool

// Options 配置并行遍历器
type Options struct {
	Root       string
	Workers    int
	FileFilter FileFilter
}

// Walker 封装通用的并行文件遍历
type Walker struct {
	root    string
	workers int
	filter  FileFilter
}

// New 创建 Walker
func New(opts Options) (*Walker, error) {
	if opts.Root == "" {
		return nil, errors.New("root 路径不能为空")
	}

	workers := opts.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
		if workers < 1 {
			workers = 1
		}
	}

	return &Walker{
		root:    opts.Root,
		workers: workers,
		filter:  opts.FileFilter,
	}, nil
}

// Walk 并行遍历文件并执行处理器
func (w *Walker) Walk(ctx context.Context, processor FileProcessor) error {
	if processor == nil {
		return errors.New("processor 不能为空")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type fileTask struct {
		path string
		info os.FileInfo
	}

	tasks := make(chan fileTask)

	var wg sync.WaitGroup
	var workerErr error
	var once sync.Once

	setErr := func(err error) {
		if err == nil {
			return
		}
		once.Do(func() {
			workerErr = err
			cancel()
		})
	}

	for i := 0; i < w.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range tasks {
				if ctx.Err() != nil {
					return
				}
				if err := processor(ctx, task.path, task.info); err != nil {
					setErr(err)
					return
				}
			}
		}()
	}

	walkErr := filepath.Walk(w.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if w.filter != nil && !w.filter(path, info) {
			return nil
		}

		select {
		case tasks <- fileTask{path: path, info: info}:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	close(tasks)
	wg.Wait()

	if workerErr != nil {
		return workerErr
	}

	if walkErr != nil {
		return walkErr
	}

	return ctx.Err()
}
