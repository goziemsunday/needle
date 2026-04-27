package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	// accept positional args
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Need two parameters: pattern and file path")
		os.Exit(1)
	}

	// extract args pattern and file path
	pattern, path := os.Args[1], os.Args[2]

	// open file from file path and handle error
	file, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	lineNumber := 0

	// scan the file, and return the lines that match the pattern
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineNumber++
		if strings.Contains(scanner.Text(), pattern) {
			fmt.Println(lineNumber, scanner.Text())
		}
	}
  // return err if an error occurs
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "scanning file", err)
		os.Exit(1)
	}
}
