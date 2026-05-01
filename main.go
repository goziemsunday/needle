package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/chiagxziem/needle/internal/search"
)

func main() {
	// define flags
	ignoreCase := flag.Bool("i", false, "ignore case distinctions in patterns")
	showLineNumbers := flag.Bool("n", false, "print line number with output lines")
	printCountPerFIle := flag.Bool("c", false, "print only a count of matching lines per file")
	printFilesWithMatches := flag.Bool("l", false, "print only filenames with matches")
	useFixedStrings := flag.Bool("F", false, "use patterns as strings instead of regular expressions")
	// parse the command line into the defined flags
	flag.Parse()

	// show usage & help message if no pattern is passed
	if len(flag.Args()) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: needle [OPTION]... PATTERNS [FILE]...")
		fmt.Println("Try 'needle --help' for more information.")
		os.Exit(1)
	}

	// get pattern and paths, if given
	pattern, paths := flag.Arg(0), flag.Args()[1:]

	// define opts from flags
	opts := search.Options{
		IgnoreCase:            *ignoreCase,
		ShowLineNumbers:       *showLineNumbers,
		PrintCountPerFile:     *printCountPerFIle,
		PrintFilesWithMatches: *printFilesWithMatches,
		UseFixedStrings:       *useFixedStrings,
	}

	// Stdin mode
	if len(paths) == 0 {
		hasMatch, err := search.SearchStdin(pattern, opts)

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if !hasMatch {
			os.Exit(1)
		}

		return
	}

	// file mode
	multipleFiles := len(paths) > 1
	hasAnyMatch := false

	for _, p := range paths {
		result, err := search.SearchFile(p, pattern, opts)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if result.HasMatch {
			hasAnyMatch = true
		}

		if opts.PrintFilesWithMatches {
			if result.HasMatch {
				fmt.Println(p)
			}
		} else if opts.PrintCountPerFile {
			if multipleFiles {
				fmt.Printf("%s:%d\n", p, result.Count)
			} else {
				fmt.Println(result.Count)
			}
		} else {
			for _, m := range result.Matches {
				if multipleFiles {
					fmt.Printf("%s:%s\n", p, m.Format(opts.ShowLineNumbers))
				} else {
					fmt.Println(m.Format(opts.ShowLineNumbers))
				}
			}
		}
	}

	// if no file matches the pattern, exit the program with code 1
	if !hasAnyMatch {
		os.Exit(1)
	}

}
