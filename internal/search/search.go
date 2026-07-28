package search

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

func compilePattern(patterns []string, opts Options) (*regexp.Regexp, error) {
	var compiled []string
	for _, p := range patterns {
		// escape all regexp metacharacters when -F is passed
		if opts.UseFixedStrings {
			p = regexp.QuoteMeta(p)
		}
		// prefix pattern with regexp for case-insensitive matching
		if opts.IgnoreCase {
			p = "(?i)" + p
		}
		// wrap pattern with `\b` for matching whole words
		if opts.WordBoundary {
			p = fmt.Sprintf(`\b%s\b`, p)
		}
		compiled = append(compiled, p)
	}

	// compile combined pattern into regexp object
	combined := strings.Join(compiled, "|")
	return regexp.Compile(combined)
}

func SearchStdin(
	ctx context.Context,
	patterns []string,
	opts Options,
	onMatch func(Match, *regexp.Regexp) bool,
) (Result, error) {
	// get regexp object from pattern and opts
	re, err := compilePattern(patterns, opts)
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

			// if -q is passed, break the loop
			if opts.Quiet {
				break
			}

			// if the callback returns false, break the loop
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
		Patterns:      patterns,
	}, scanner.Err()
}

func Search(
	ctx context.Context,
	r io.Reader,
	path string,
	patterns []string,
	opts Options,
) (Result, error) {
	// get regexp object from pattern and opts
	re, err := compilePattern(patterns, opts)
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

			if opts.Quiet {
				break
			}
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
		Patterns:      patterns,
	}, nil
}
