package search

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

type workerResult struct {
	Path   string
	Result Result
	Err    error
}

func SearchFile(
	ctx context.Context,
	path string,
	patterns []string,
	opts Options,
) (Result, error) {
	// return immediately if the search was cancelled
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

	// check if path is a directory
	info, err := os.Stat(path)
	if err != nil {
		return Result{}, err
	}
	// if dir, return detailed error
	if info.IsDir() {
		return Result{}, fmt.Errorf("%s: is a directory (use -r for recursive search)", path)
	}

	// open file from file path and handle error
	file, err := os.Open(path)
	if err != nil {
		return Result{}, err
	}
	defer file.Close()

	// read first 512 bytes to check for binary
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return Result{}, err
	}

	// if binary file, return empty result quietly
	if bytes.IndexByte(buf[:n], 0) != -1 {
		return Result{}, nil
	}

	// stitch the already-read bytes with the rest of the file
	r := io.MultiReader(bytes.NewReader(buf[:n]), file)

	return Search(ctx, r, path, patterns, opts)
}

func searchPaths(
	ctx context.Context,
	cancel context.CancelFunc,
	paths []string,
	patterns []string,
	opts Options,
) ([]Result, error) {
	numPaths := len(paths)
	pathsChan := make(chan string, numPaths)
	resultsChan := make(chan workerResult, numPaths)

	workerCount := runtime.NumCPU()

	// start workers
	var wg sync.WaitGroup
	for range workerCount {
		wg.Go(func() {
			searchFileWorker(ctx, cancel, pathsChan, resultsChan, patterns, opts)
		})
	}

	// close results channel once all workers finish
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// send paths for processing by the workers
	for _, p := range paths {
		pathsChan <- p
	}
	close(pathsChan)

	// collect and handle results
	var results []Result
	var errs []error
	for r := range resultsChan {
		if err := r.Err; err != nil {
			if errors.Is(err, context.Canceled) {
				// expected in workers when -q is passed
				// done to prevent printing an error msg when context.Canceled occurs
				continue
			}
			errs = append(errs, fmt.Errorf("needle: %s: %v", r.Path, err))
			continue
		}

		results = append(results, r.Result)
	}

	if len(errs) > 0 {
		// partial results are still valid, the caller decides the exit status
		return results, errors.Join(errs...)
	}

	return results, nil
}

func searchFileWorker(
	ctx context.Context,
	cancel context.CancelFunc,
	paths <-chan string,
	workerResults chan<- workerResult,
	patterns []string,
	opts Options,
) {
	for {
		select {
		case <-ctx.Done():
			// worker cancelled while waiting for job
			return

		case path, ok := <-paths:
			// check if the jobs channel is closed
			if !ok {
				// jobs channel closed, exiting
				return
			}

			// pass path from jobs channel to search file
			result, err := SearchFile(ctx, path, patterns, opts)

			// if -q is passed and a match is found, cancel all other worker
			if opts.Quiet && result.HasMatch {
				cancel()
				return
			}

			select {
			case <-ctx.Done():
				// worker cancelled after picking up the path; dropping result
				return

			case workerResults <- workerResult{Path: path, Result: result, Err: err}:
			}
		}
	}
}

func SearchDir(
	ctx context.Context,
	cancel context.CancelFunc,
	root string,
	patterns []string,
	opts Options,
) ([]Result, error) {
	// return immediately if the search was cancelled
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

	return searchPaths(ctx, cancel, paths, patterns, opts)
}
