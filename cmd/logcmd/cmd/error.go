package cmd

import (
	"errors"
	"fmt"
)

// ExitError 表示带退出码的错误。
type ExitError struct {
	err  error
	code int
}

// Error 实现 error 接口。
func (e *ExitError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

// ExitCode 返回建议的进程退出码。
func (e *ExitError) ExitCode() int {
	if e == nil {
		return 0
	}
	return e.code
}

func newExitError(err error, code int) *ExitError {
	if err == nil {
		err = errors.New("")
	}
	return &ExitError{err: err, code: code}
}

func newExitErrorf(code int, format string, args ...interface{}) *ExitError {
	return newExitError(fmt.Errorf(format, args...), code)
}
