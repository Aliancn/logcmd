package cliutil

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// ConfirmationPrompt 描述一次交互式确认所需的信息。
type ConfirmationPrompt struct {
	Message       string // 主提示语
	AcceptedWords []string
	RejectedWords []string
	DefaultYes    bool
	Reader        io.Reader
	Writer        io.Writer
}

// Confirm 执行一次交互式确认，返回用户是否同意。
func Confirm(prompt ConfirmationPrompt) (bool, error) {
	reader := prompt.reader()
	writer := prompt.writer()

	fmt.Fprint(writer, prompt.Message)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	clean := strings.TrimSpace(strings.ToLower(input))
	if clean == "" {
		return prompt.DefaultYes, nil
	}

	if wordIn(clean, prompt.AcceptedWords) {
		return true, nil
	}
	if wordIn(clean, prompt.RejectedWords) {
		return false, nil
	}

	// 未知输入视为拒绝，以避免误删除。
	return false, nil
}

func (p ConfirmationPrompt) reader() *bufio.Reader {
	if p.Reader != nil {
		if r, ok := p.Reader.(*bufio.Reader); ok {
			return r
		}
		return bufio.NewReader(p.Reader)
	}
	return bufio.NewReader(os.Stdin)
}

func (p ConfirmationPrompt) writer() io.Writer {
	if p.Writer != nil {
		return p.Writer
	}
	return os.Stdout
}

func wordIn(value string, list []string) bool {
	for _, item := range list {
		if strings.TrimSpace(strings.ToLower(item)) == value {
			return true
		}
	}
	return false
}
