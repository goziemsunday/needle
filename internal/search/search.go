package search

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
)

type Options struct {
	IgnoreCase            bool
	ShowLineNumbers       bool
	PrintCountPerFile     bool
	PrintFilesWithMatches bool
	UseFixedStrings       bool
	RecursiveSearch       bool
	Include               string
	Exclude               string
	ExcludeDir            string
}

type Match struct {
	LineNumber int
	Line       string
}

type Formatter struct {
	Highlight func(a ...any) string
	LineNum   func(a ...any) string
	Sep       func(a ...any) string
}

func (m Match) Format(re *regexp.Regexp, f Formatter, opts Options) string {
	highlighted := re.ReplaceAllStringFunc(m.Line, func(s string) string {
		return f.Highlight(s)
	})
	if opts.ShowLineNumbers {
		return fmt.Sprintf("%s%s%s", f.LineNum(m.LineNumber), f.Sep(":"), highlighted)
	}
	return highlighted
}

type Result struct {
	Path          string
	Matches       []Match
	Count         int
	HasMatch      bool
	RegexpPattern *regexp.Regexp
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

func SearchStdin(
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

		if re.MatchString(line) {
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
		Path:          path,
		Matches:       matches,
		Count:         len(matches),
		HasMatch:      len(matches) > 0,
		RegexpPattern: re,
	}, nil
}

func fileMatchesFilters(name string, opts Options) (bool, error) {
	// if --include is set, skip files that don't match the glob
	if opts.Include != "" {
		matched, err := filepath.Match(opts.Include, name)
		if err != nil {
			return false, err
		}
		if !matched {
			return false, nil
		}
	}

	// if --exclude is set, skip files that match the glob
	if opts.Exclude != "" {
		matched, err := filepath.Match(opts.Exclude, name)
		if err != nil {
			return false, err
		}
		if matched {
			return false, nil
		}
	}

	return true, nil
}

func SearchFile(
	ctx context.Context,
	path, pattern string,
	opts Options,
) (Result, error) {
	select {
	case <-ctx.Done():
		return Result{}, ctx.Err()
	default:
	}

	// ensure path matches include/exclude filters if given
	ok, err := fileMatchesFilters(filepath.Base(path), opts)
	if err != nil {
		return Result{}, err
	}
	if !ok {
		return Result{}, nil
	}

	// open file from file path and handle error
	file, err := os.Open(path)
	if err != nil {
		return Result{}, err
	}
	// close file after function runs
	defer file.Close()

	// read first 512 bytes to check for binary
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return Result{}, err
	}

	if bytes.IndexByte(buf[:n], 0) != -1 {
		// binary file, return empty result quietly
		return Result{}, nil
	}

	// stitch the already-read bytes with the rest of the file
	r := io.MultiReader(bytes.NewReader(buf[:n]), file)

	return Search(r, path, pattern, opts)
}

type workerResult struct {
	Path   string
	Result Result
	Err    error
}

func searchFileWorker(
	ctx context.Context,
	jobs <-chan string,
	workerResults chan<- workerResult,
	pattern string,
	opts Options,
) {
	for {
		select {
		case <-ctx.Done():
			// worker cancelled while waiting for job
			return

		case path, ok := <-jobs:
			// check if the jobs channel is closed
			if !ok {
				// jobs channel closed, exiting
				return
			}

			// pass path from jobs channel to search file
			result, err := SearchFile(ctx, path, pattern, opts)

			select {
			case <-ctx.Done():
				// worker cancelled after picking up the path; dropping result
				return

			case workerResults <- workerResult{Path: path, Result: result, Err: err}:
			}
		}
	}
}

func searchPaths(
	ctx context.Context,
	paths []string,
	pattern string,
	opts Options,
) ([]Result, error) {
	numPaths := len(paths)
	pathChan := make(chan string, numPaths)
	resultsChan := make(chan workerResult, numPaths)

	workerCount := runtime.NumCPU()

	// start workers
	var wg sync.WaitGroup
	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			searchFileWorker(ctx, pathChan, resultsChan, pattern, opts)
		}()
	}

	// send jobs into the workers, and close jobs channel
	for _, p := range paths {
		pathChan <- p
	}
	close(pathChan)

	go func() {
		// start waiter concurrently with workers so main can collect results
		wg.Wait()
		// close the results channel once the workers all exit
		close(resultsChan)
	}()

	// collect and handle results
	var results []Result
	for r := range resultsChan {
		if err := r.Err; err != nil {
			fmt.Fprintf(os.Stderr, "needle: %s: %v\n", r.Path, err)
			continue
		}

		results = append(results, r.Result)
	}

	return results, nil
}

func SearchDir(ctx context.Context, root, pattern string, opts Options) ([]Result, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	var paths []string

	// traverse through the given directory
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// dirs to skip
		if d.IsDir() && d.Name() != "." {
			// skip hidden dirs
			if d.Name()[0] == '.' {
				return filepath.SkipDir
			}

			// skip excluded dirs when --exclude-dir is set
			if opts.ExcludeDir != "" {
				matched, err := filepath.Match(opts.ExcludeDir, d.Name())
				if err != nil {
					return err
				}
				if matched {
					return filepath.SkipDir
				}
			}
		}

		if !d.IsDir() {
			paths = append(paths, path)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return searchPaths(ctx, paths, pattern, opts)
}
