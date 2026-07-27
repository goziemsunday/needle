package search

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
)

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

func SearchStdin(
	ctx context.Context,
	pattern string,
	opts Options,
	onMatch func(Match, *regexp.Regexp) bool,
) (Result, error) {
	// get regexp object from pattern and opts
	re, err := compilePattern(pattern, opts)
	if err != nil {
		return Result{}, fmt.Errorf("invalid pattern: %w", err)
	}

	scanner := bufio.NewScanner(os.Stdin)
	lineNumber := 0
	var matches []Match

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		matched := re.MatchString(line)

		if opts.InvertMatch {
			matched = !matched
		}

		if matched {
			m := Match{lineNumber, line}
			matches = append(matches, m)
			if !onMatch(m, re) {
				break
			}
		}
	}

	return Result{
		Matches:       matches,
		Count:         len(matches),
		HasMatch:      len(matches) > 0,
		RegexpPattern: re,
	}, scanner.Err()
}

func Search(
	ctx context.Context,
	r io.Reader,
	path, pattern string,
	opts Options,
) (Result, error) {
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
		matched := re.MatchString(line)

		if opts.InvertMatch {
			matched = !matched
		}

		if matched {
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
		Path:          path,
		Matches:       matches,
		Count:         len(matches),
		HasMatch:      len(matches) > 0,
		RegexpPattern: re,
	}, nil
}
