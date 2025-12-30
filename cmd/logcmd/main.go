package main

import (
	"fmt"
	"os"

	"github.com/aliancn/logcmd/cmd/logcmd/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		if exitErr, ok := err.(interface{ ExitCode() int }); ok {
			if err.Error() != "" {
				fmt.Fprintln(os.Stderr, err.Error())
			}
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
