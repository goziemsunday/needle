package main

import (
	"errors"
	"os"

	"github.com/goziemsunday/needle/cmd"
)

func main() {
	if err := cmd.Run(); err != nil {
		if _, ok := errors.AsType[cmd.UsageError](err); ok {
			os.Exit(2)
		}
		os.Exit(1)
	}
}
