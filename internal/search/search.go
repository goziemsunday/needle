package search

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

func SearchStdin(
	ctx context.Context,
	patterns []string,
	opts Options,
	onMatch func(Match, *regexp.Regexp) bool,
	onContextLine func(ContextLine) bool,
) (Result, error) {
	// get regexp object from pattern and opts
	re, err := compilePattern(patterns, opts)
	if err != nil {
		return Result{}, fmt.Errorf("invalid pattern: %w", err)
	}

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(scanLinesKeepCR)
	// raise the 64KB scanner cap
	scanner.Buffer(make([]byte, 64*1024), 1<<30)
	lineNumber := 0
	isBinary := false
	var matches []Match

	ring := newLineRing(opts.BeforeContext + 1)
	afterRemaining := 0

	for scanner.Scan() {
		lineNumber++
		raw := scanner.Bytes()
		line := string(raw)
		ring.add(line, lineNumber)

		// a NUL byte anywhere in the input marks it binary; grep skips
		// per-line output for binary input, so the line-printing
		// callbacks are skipped once the first NUL is seen while the
		// scan keeps running for counts
		if bytes.IndexByte(raw, 0) != -1 {
			isBinary = true
		}

		matched := re.MatchString(line)
		if opts.InvertMatch {
			matched = !matched
		}

		if matched {
			// a matching line starts a fresh after-window
			afterRemaining = opts.AfterContext
			m := Match{LineNumber: lineNumber, Line: line}

			if opts.BeforeContext > 0 {
				m.Before = ring.beforeContext()
			}

			matches = append(matches, m)

			// if -q is passed, break the loop
			if opts.Quiet {
				break
			}

			// if the callback returns false, break the loop; binary
			// input skips line printing unless -l needs the name
			if !isBinary || opts.PrintFilesWithMatches {
				if !onMatch(m, re) {
					break
				}
			}

		} else if afterRemaining > 0 {
			// non-matching line within the last match's after-window
			last := &matches[len(matches)-1]
			ctxLine := ContextLine{Number: lineNumber, Text: line}
			last.After = append(last.After, ctxLine)
			afterRemaining--
			if !isBinary && !onContextLine(ctxLine) {
				break
			}
		}
	}

	return Result{
		Matches:       matches,
		Count:         len(matches),
		HasMatch:      len(matches) > 0,
		IsBinary:      isBinary,
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
	scanner.Split(scanLinesKeepCR)
	// raise the 64KB scanner cap
	scanner.Buffer(make([]byte, 64*1024), 1<<30)
	lineNumber := 0
	isBinary := false
	var matches []Match

	ring := newLineRing(opts.BeforeContext + 1)
	afterRemaining := 0

	// scan the file, and get matches if any
	for scanner.Scan() {
		lineNumber++
		raw := scanner.Bytes()
		line := string(raw)
		ring.add(line, lineNumber)

		// a NUL byte anywhere in the file marks it binary; the file is
		// still scanned (exit status follows the match) but the caller
		// suppresses per-line output
		if bytes.IndexByte(raw, 0) != -1 {
			isBinary = true
		}

		matched := re.MatchString(line)
		if opts.InvertMatch {
			matched = !matched
		}

		if matched {
			// a match starts a fresh after-context window
			afterRemaining = opts.AfterContext
			m := Match{LineNumber: lineNumber, Line: line}

			if opts.BeforeContext > 0 {
				m.Before = ring.beforeContext()
			}

			matches = append(matches, m)

			// if -q is passed, break the loop
			if opts.Quiet {
				break
			}
		} else if afterRemaining > 0 {
			// non-matching line within the last match's after-window
			last := &matches[len(matches)-1]
			last.After = append(last.After, ContextLine{Number: lineNumber, Text: line})
			afterRemaining--
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
		IsBinary:      isBinary,
		RegexpPattern: re,
		Patterns:      patterns,
	}, nil
}

// compiles the combined patterns into a single regexp,
// exported so callers can validate patterns upfront
func CompilePatterns(patterns []string, opts Options) (*regexp.Regexp, error) {
	return compilePattern(patterns, opts)
}

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

type lineRing struct {
	lines    []string
	lineNums []int
	head     int
	count    int
	size     int
}

func newLineRing(size int) lineRing {
	return lineRing{
		lines:    make([]string, size),
		lineNums: make([]int, size),
		size:     size,
	}
}

// add writes a new line into the ring, overwriting the oldest when full
func (l *lineRing) add(line string, num int) {
	if l.size == 0 {
		return
	}
	l.lines[l.head] = line
	l.lineNums[l.head] = num
	l.head = (l.head + 1) % l.size // head never exceeds size
	if l.count < l.size {
		l.count++
	}
}

// beforeContext returns every line in the ring except the most recent
// one (which is the current match line), oldest first
func (l *lineRing) beforeContext() []ContextLine {
	n := l.count - 1
	if n <= 0 {
		return nil
	}
	start := (l.head - l.count + l.size) % l.size
	ctx := make([]ContextLine, 0, n)
	for i := range n {
		idx := (start + i) % l.size
		ctx = append(ctx, ContextLine{Number: l.lineNums[idx], Text: l.lines[idx]})
	}
	return ctx
}

// scanLinesKeepCR splits on '\n' only; a trailing '\r' stays part of the
// line, matching grep's line handling
func scanLinesKeepCR(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		return i + 1, data[:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}
