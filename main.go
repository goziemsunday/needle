package main

import (
	"os"

	"github.com/goziemsunday/needle/cmd"
)

func main() {
	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
}
