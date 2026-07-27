package cmd

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/goziemsunday/needle/internal/output"
	"github.com/goziemsunday/needle/internal/search"
	"github.com/spf13/pflag"
)

func Run() error {
	// create context with cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// define flags
	ignoreCase := pflag.BoolP("ignore-case", "i", false, "ignore case distinctions in patterns")
	showLineNumbers := pflag.BoolP("line-number", "n", false, "print line number with output lines")
	printCountPerFile := pflag.BoolP("count", "c", false, "print only a count of matching lines per file")
	printFilesWithMatches := pflag.BoolP("files-with-matches", "l", false, "print only names of files with matches")
	recursiveSearch := pflag.BoolP("recursive", "r", false, "search files & directories recursively")
	useFixedStrings := pflag.BoolP("fixed-strings", "F", false, "use patterns as strings instead of regular expressions")
	invertMatch := pflag.BoolP("invert-match", "v", false, "print lines that do not match the pattern")
	include := pflag.String("include", "", "search only files matching glob e.g. '*.go'")
	exclude := pflag.String("exclude", "", "skip files that match glob e.g. '*.go'")
	excludeDir := pflag.String("exclude-dir", "", "skip directories matching glob e.g. 'vendor'")
	noColor := pflag.Bool("no-color", false, "never highlight the matching strings")

	// parse the command line into the defined flags
	pflag.Parse()

	// show usage & help message if no pattern is passed
	if len(pflag.Args()) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: needle [OPTION]... PATTERNS [FILE]...")
		fmt.Fprintln(os.Stderr, "Try 'needle --help' for more information.")
		return fmt.Errorf("no pattern passed")
	}

	// get pattern and paths, if given
	pattern, paths := pflag.Arg(0), pflag.Args()[1:]

	// define opts from flags
	opts := search.Options{
		IgnoreCase:            *ignoreCase,
		ShowLineNumbers:       *showLineNumbers,
		PrintCountPerFile:     *printCountPerFile,
		PrintFilesWithMatches: *printFilesWithMatches,
		UseFixedStrings:       *useFixedStrings,
		RecursiveSearch:       *recursiveSearch,
		InvertMatch:           *invertMatch,
		Include:               *include,
		Exclude:               *exclude,
		ExcludeDir:            *excludeDir,
	}

	// enable no-color mode if stdout is not a terminal
	output.SetupColors(noColor)

	// init variable to track discovery of a match
	hasAnyMatch := false

	// RECURSIVE MODE
	if opts.RecursiveSearch {
		var roots []string
		if len(paths) == 0 {
			roots = append(roots, ".")
		} else {
			roots = paths
		}

		for _, root := range roots {
			results, err := search.SearchDir(ctx, root, pattern, opts)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return err
			}

			for _, result := range results {
				if result.HasMatch {
					hasAnyMatch = true
				}

				output.GetOutput(result, opts, true)
			}
		}

		// if no file matches the pattern, exit the program with code 1
		if !hasAnyMatch {
			return fmt.Errorf("no match found")
		}

		return nil
	}

	// STDIN MODE
	if len(paths) == 0 {
		result, err := search.SearchStdin(
			ctx, pattern, opts,
			func(m search.Match, r *regexp.Regexp) bool {
				// handle -l immediately if passed
				if opts.PrintFilesWithMatches {
					fmt.Println(output.Magenta("(standard input)"))
					return false
				}
				// if there's no -c, handle normally
				if !opts.PrintCountPerFile {
					fmt.Println(output.FormatMatch(m, r, output.DefaultFormatter, opts))
				}
				return true
			},
		)

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return err
		}
		if !result.HasMatch {
			return fmt.Errorf("no match found")
		}

		// handle -c after all input is read
		if opts.PrintCountPerFile && !opts.PrintFilesWithMatches {
			fmt.Println(result.Count)
		}

		return nil
	}

	// FILE MODE
	multipleFiles := len(paths) > 1

	for _, p := range paths {
		result, err := search.SearchFile(ctx, p, pattern, opts)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return err
		}

		if result.HasMatch {
			hasAnyMatch = true
		}

		output.GetOutput(result, opts, multipleFiles)
	}

	// if no file matches the pattern, exit the program with code 1
	if !hasAnyMatch {
		return fmt.Errorf("no match found")
	}

	return nil
}
