package search

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
)

type Options struct {
	IgnoreCase            bool
	ShowLineNumbers       bool
	PrintCountPerFile     bool
	PrintFilesWithMatches bool
	UseFixedStrings       bool
}

type Match struct {
	LineNumber int
	Line       string
}

func (m Match) Format(showLineNumbers bool) string {
	if showLineNumbers {
		return fmt.Sprintf("%d: %s", m.LineNumber, m.Line)
	}
	return m.Line
}

type Result struct {
	Matches  []Match
	Count    int
	HasMatch bool
}

func Search(path, pattern string, opts Options) (Result, error) {
	// open file from file path and handle error
	file, err := os.Open(path)
	if err != nil {
		return Result{}, err
	}
	// close file after function runs
	defer file.Close()

	// escape all regexp metacharacters when -F is passed
	if opts.UseFixedStrings {
		pattern = regexp.QuoteMeta(pattern)
	}
	// prefix pattern with regexp for case-insensitive matching
	if opts.IgnoreCase {
		pattern = "(?i)" + pattern
	}
	// compile pattern into regexp object
	re, err := regexp.Compile(pattern)
	if err != nil {
		return Result{}, fmt.Errorf("invalid pattern: %w", err)
	}

	// create scanner
	scanner := bufio.NewScanner(file)

	lineNumber := 0
	var matches []Match

	// scan the file, and get matches if any
	for scanner.Scan() {
		lineNumber++

		if re.MatchString(scanner.Text()) {
			matches = append(matches, Match{
				LineNumber: lineNumber,
				Line:       scanner.Text(),
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return Result{}, err
	}

	return Result{
		Matches:  matches,
		Count:    len(matches),
		HasMatch: len(matches) > 0,
	}, nil
}
