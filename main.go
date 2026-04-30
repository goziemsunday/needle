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

	if len(flag.Args()) < 2 {
		fmt.Fprintln(os.Stderr, "Need two parameters: pattern and file path")
		os.Exit(1)
	}
	pattern, path := flag.Arg(0), flag.Arg(1)

	// define opts from flags
	opts := search.Options{
		IgnoreCase:            *ignoreCase,
		ShowLineNumbers:       *showLineNumbers,
		PrintCountPerFile:     *printCountPerFIle,
		PrintFilesWithMatches: *printFilesWithMatches,
		UseFixedStrings:       *useFixedStrings,
	}

	// perform search
	result, err := search.Search(path, pattern, opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if opts.PrintFilesWithMatches {
		if result.HasMatch {
			fmt.Println(path)
		} else {
			os.Exit(1)
		}
	} else if opts.PrintCountPerFile {
		fmt.Printf("%d\n", result.Count)
		if !result.HasMatch {
			os.Exit(1)
		}
	} else {
		// if there are no matches (and no errors) exit program
		if !result.HasMatch {
			os.Exit(1)
		}
		// print matches to os.Stdout
		for _, m := range result.Matches {
			fmt.Println(m.Format(opts.ShowLineNumbers))
		}
	}
}
