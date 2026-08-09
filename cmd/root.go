package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"

	"github.com/goziemsunday/needle/internal/output"
	"github.com/goziemsunday/needle/internal/search"
	"github.com/spf13/pflag"
)

type UsageError struct{ Msg string }

func (e UsageError) Error() string { return e.Msg }

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
	quiet := pflag.BoolP("quiet", "q", false, "suppress all output, exit immediately on first match")
	wordBoundary := pflag.BoolP("word-regexp", "w", false, "matches only whole words")
	include := pflag.String("include", "", "search only files matching glob e.g. '*.go'")
	exclude := pflag.String("exclude", "", "skip files that match glob e.g. '*.go'")
	excludeDir := pflag.String("exclude-dir", "", "skip directories matching glob e.g. 'vendor'")
	beforeContext := pflag.IntP("before-context", "B", 0, "print N lines before each match")
	afterContext := pflag.IntP("after-context", "A", 0, "print N lines after each match")
	fullContext := pflag.IntP("context", "C", 0, "print N lines before and after each match")
	groupSep := pflag.String("group-separator", "--", "separator between context groups")

	// multiple patterns
	var extraPatterns []string
	pflag.StringArrayVarP(&extraPatterns, "regexp", "e", nil, "use pattern for matching (can be repeated)")

	// color
	colorWhen := "auto"
	pflag.StringVar(&colorWhen, "color", "auto", "use markers to highlight the matching strings; WHEN is 'always', 'never', or 'auto'")
	pflag.StringVar(&colorWhen, "colour", "auto", "use markers to highlight the matching strings; WHEN is 'always', 'never', or 'auto'")
	// allow bare `--color`/`--colour`
	pflag.Lookup("color").NoOptDefVal = "auto"
	pflag.Lookup("colour").NoOptDefVal = "auto"

	// parse the command line into the defined flags
	pflag.Parse()

	// validate colorWhen
	whenColor := colorWhen
	if whenColor != "auto" && whenColor != "always" && whenColor != "never" {
		fmt.Fprintf(os.Stderr, "needle: invalid argument %q for '--color'; valid values are 'always', 'never', or 'auto'\n", whenColor)
		fmt.Fprintln(os.Stderr, "Usage: needle [OPTION]... PATTERN [FILE]...")
		return UsageError{Msg: "invalid argument for --color"}
	}

	// get patterns and paths, if given
	var patterns []string
	var paths []string
	if len(extraPatterns) > 0 {
		// -e patterns provided, all positional args are paths
		paths = pflag.Args()
	} else if len(pflag.Args()) > 0 {
		// no -e patterns, first positional arg is the pattern
		patterns = append(patterns, pflag.Arg(0))
		paths = pflag.Args()[1:]
	}
	patterns = append(patterns, extraPatterns...)

	// show usage & help message if no pattern is passed
	if len(patterns) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: needle [OPTION]... PATTERN [FILE]...")
		fmt.Fprintln(os.Stderr, "Try 'needle --help' for more information.")
		return UsageError{Msg: "no pattern passed"}
	}

	// define opts from flags
	opts := search.Options{
		IgnoreCase:            *ignoreCase,
		ShowLineNumbers:       *showLineNumbers,
		PrintCountPerFile:     *printCountPerFile,
		PrintFilesWithMatches: *printFilesWithMatches,
		UseFixedStrings:       *useFixedStrings,
		RecursiveSearch:       *recursiveSearch,
		InvertMatch:           *invertMatch,
		Quiet:                 *quiet,
		WordBoundary:          *wordBoundary,
		Include:               *include,
		Exclude:               *exclude,
		ExcludeDir:            *excludeDir,
		BeforeContext:         *beforeContext,
		AfterContext:          *afterContext,
		GroupSeparator:        *groupSep,
	}

	// -C overrides both -A and -B
	if *fullContext > 0 {
		opts.BeforeContext = *fullContext
		opts.AfterContext = *fullContext
	}

	// validate the pattern compiles upfront
	if _, err := search.CompilePatterns(patterns, opts); err != nil {
		fmt.Fprintf(os.Stderr, "needle: invalid pattern: %v\n", err)
		return UsageError{Msg: "invalid pattern"}
	}

	// color highlighting: "auto" = only when stdout is a terminal (default)
	// "always" = force color even when piped (e.g. `less -R`), "never" = disable
	// bare --color behaves as "auto" via NoOptDefVal
	output.SetupColors(colorWhen)

	// init variables to track discovery of a match and errors
	hasAnyMatch := false
	anyError := false

	// RECURSIVE MODE
	if opts.RecursiveSearch {
		var roots []string
		if len(paths) == 0 {
			roots = append(roots, ".")
		} else {
			roots = paths
		}

		for _, root := range roots {
			results, err := search.SearchDir(ctx, cancel, root, patterns, opts)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					// exit 0 when -q is passed and context.Canceled occurs
					hasAnyMatch = true
					break
				}
				fmt.Fprintln(os.Stderr, err)
				// partial results may still be valid (per-file errors)
				anyError = true
			}

			// match found by worker, context was cancelled
			if ctx.Err() != nil {
				hasAnyMatch = true
				break
			}

			multipleFiles := len(results) > 1
			for _, result := range results {
				if result.HasMatch {
					hasAnyMatch = true
				}

				output.GetOutput(result, opts, multipleFiles)
			}
		}

		// -q and a match found, exit 0 regardless of errors
		if opts.Quiet && hasAnyMatch {
			return nil
		}

		if anyError {
			return UsageError{Msg: "errors encountered during search"}
		}

		// if no file matches the pattern, exit the program with code 1
		if !hasAnyMatch {
			return fmt.Errorf("no match found")
		}

		return nil
	}

	// STDIN MODE
	if len(paths) == 0 {
		printer := output.NewMatchPrinter(opts, "", false)

		result, err := search.SearchStdin(
			ctx, patterns, opts,
			func(m search.Match, r *regexp.Regexp) bool {
				// handle -l immediately if passed
				if opts.PrintFilesWithMatches {
					fmt.Println(output.Magenta("(standard input)"))
					return false
				}
				// skip per-line printing for -c (count printed at the end)
				if !opts.PrintCountPerFile {
					printer.Print(m, r)
				}
				return true
			},
			func(c search.ContextLine) bool {
				if !opts.PrintCountPerFile {
					printer.PrintContextLine(c)
				}
				return true
			},
		)

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			// -q with a match already found still exits 0
			if opts.Quiet && result.HasMatch {
				return nil
			}
			return UsageError{Msg: "error while reading from standard input"}
		}
		if !result.HasMatch {
			return fmt.Errorf("no match found")
		}

		// handle -c after all input is read
		if opts.PrintCountPerFile && !opts.PrintFilesWithMatches && !opts.Quiet {
			fmt.Println(result.Count)
		}

		return nil
	}

	// FILE MODE
	multipleFiles := len(paths) > 1

	for _, p := range paths {
		result, err := search.SearchFile(ctx, p, patterns, opts)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			anyError = true
			continue
		}

		if result.HasMatch {
			hasAnyMatch = true
		}

		// if -q is passed, break the loop once a match is found
		if opts.Quiet && hasAnyMatch {
			break
		}

		output.GetOutput(result, opts, multipleFiles)
	}

	// -q and a match found, exit 0 regardless of errors
	if opts.Quiet && hasAnyMatch {
		return nil
	}

	if anyError {
		return UsageError{Msg: "errors encountered during search"}
	}

	if !hasAnyMatch {
		return fmt.Errorf("no match found")
	}

	return nil
}
