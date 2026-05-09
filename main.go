package main

import (
	"fmt"
	"os"
	"regexp"

	"github.com/chiagxziem/needle/internal/search"
	"github.com/fatih/color"
	"github.com/spf13/pflag"
	"golang.org/x/term"
)

var (
	magenta   = color.New(color.FgMagenta).SprintFunc()
	green     = color.New(color.FgGreen).SprintFunc()
	red       = color.New(color.FgRed, color.Bold).SprintFunc()
	formatter = search.Formatter{
		Highlight: red,
		LineNum:   green,
		Sep:       magenta,
	}
)

func main() {
	// flags
	ignoreCase := pflag.BoolP("ignore-case", "i", false, "ignore case distinctions in patterns")
	showLineNumbers := pflag.BoolP("line-number", "n", false, "print line number with output lines")
	printCountPerFile := pflag.BoolP("count", "c", false, "print only a count of matching lines per file")
	printFilesWithMatches := pflag.BoolP("files-with-matches", "l", false, "print only names of files with matches")
	recursiveSearch := pflag.BoolP("recursive", "r", false, "search files & directories recursively")
	useFixedStrings := pflag.BoolP("fixed-strings", "F", false, "use patterns as strings instead of regular expressions")
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
		os.Exit(1)
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
		Include:               *include,
		Exclude:               *exclude,
		ExcludeDir:            *excludeDir,
		NoColor:               *noColor,
	}

	// enable no-color mode if stdout is not a terminal
	if *noColor || !term.IsTerminal(int(os.Stdout.Fd())) {
		color.NoColor = true
	}

	hasAnyMatch := false

	// recursive mode
	if opts.RecursiveSearch {
		var roots []string
		if len(paths) == 0 {
			roots = append(roots, ".")
		} else {
			roots = paths
		}

		for _, root := range roots {
			results, err := search.SearchDir(root, pattern, opts)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			for _, result := range results {
				if result.HasMatch {
					hasAnyMatch = true
				}

				getOutput(result, opts, true)
			}
		}

		// if no file matches the pattern, exit the program with code 1
		if !hasAnyMatch {
			os.Exit(1)
		}
		return
	}

	// Stdin mode
	if len(paths) == 0 {
		result, err := search.SearchStdin(pattern, opts, func(m search.Match, r *regexp.Regexp) bool {
			// handle -l immediately is passed
			if opts.PrintFilesWithMatches {
				fmt.Println(magenta("(standard input)"))
				return false
			}
			// if there's no -c, handle normally
			if !opts.PrintCountPerFile {
				fmt.Println(m.Format(r, formatter, opts))
			}
			return true
		})

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if !result.HasMatch {
			os.Exit(1)
		}

		// handle -c after all input is read
		if opts.PrintCountPerFile && !opts.PrintFilesWithMatches {
			fmt.Println(result.Count)
		}

		return
	}

	// file mode
	multipleFiles := len(paths) > 1

	for _, p := range paths {
		result, err := search.SearchFile(p, pattern, opts)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if result.HasMatch {
			hasAnyMatch = true
		}

		getOutput(result, opts, multipleFiles)
	}

	// if no file matches the pattern, exit the program with code 1
	if !hasAnyMatch {
		os.Exit(1)
	}

}

func getOutput(r search.Result, opts search.Options, multipleFiles bool) {
	if opts.PrintFilesWithMatches {
		if r.HasMatch {
			fmt.Println(magenta(r.Path))
		}
	} else if opts.PrintCountPerFile {
		if multipleFiles {
			fmt.Printf("%s%s%d\n", magenta(r.Path), magenta(":"), r.Count)
		} else {
			fmt.Println(r.Count)
		}
	} else {
		for _, m := range r.Matches {
			if multipleFiles {
				fmt.Printf("%s%s%s\n", magenta(r.Path), magenta(":"), m.Format(r.RegexpPattern, formatter, opts))
			} else {
				fmt.Println(m.Format(r.RegexpPattern, formatter, opts))
			}
		}
	}
}
