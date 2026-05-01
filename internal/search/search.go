package search

import (
	"bufio"
	"fmt"
	"io"
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

func compilePattern(pattern string, opts Options) (*regexp.Regexp, error) {
	// escape all regexp metacharacters when -F is passed
	if opts.UseFixedStrings {
		pattern = regexp.QuoteMeta(pattern)
	}
	// prefix pattern with regexp for case-insensitive matching
	if opts.IgnoreCase {
		pattern = "(?i)" + pattern
	}
	// compile pattern into regexp object
	return regexp.Compile(pattern)
}

func SearchStdin(pattern string, opts Options) (bool, error) {
	// get regexp object from pattern and opts
	re, err := compilePattern(pattern, opts)
	if err != nil {
		return false, fmt.Errorf("invalid pattern: %w", err)
	}

	scanner := bufio.NewScanner(os.Stdin)
	lineNumber := 0
	hasMatch := false

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		if re.MatchString(line) {
			hasMatch = true
			fmt.Println(Match{lineNumber, line}.Format(opts.ShowLineNumbers))
		}
	}

	return hasMatch, scanner.Err()
}

func SearchFile(path, pattern string, opts Options) (Result, error) {
	// open file from file path and handle error
	file, err := os.Open(path)
	if err != nil {
		return Result{}, err
	}
	// close file after function runs
	defer file.Close()

	return Search(file, pattern, opts)
}

func Search(r io.Reader, pattern string, opts Options) (Result, error) {
	// get regexp object from pattern and opts
	re, err := compilePattern(pattern, opts)
	if err != nil {
		return Result{}, fmt.Errorf("invalid pattern: %w", err)
	}

	scanner := bufio.NewScanner(r)
	lineNumber := 0
	var matches []Match

	// scan the file, and get matches if any
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		if re.MatchString(line) {
			matches = append(matches, Match{
				LineNumber: lineNumber,
				Line:       line,
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
