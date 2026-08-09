package cmd

import "fmt"

const helpText = `Needle - a fast grep alternative

Usage:
  needle [OPTION]... PATTERN [FILE]...
  needle [OPTION]... -e PATTERN [FILE]...

Search FILEs for occurrences of PATTERN and print the matching lines.
With no FILE, or when FILE is '-', read standard input. Use -r to
search files and directories recursively.

Options:
  -A, --after-context NUM     print NUM lines after each match
  -B, --before-context NUM    print NUM lines before each match
  -c, --count                 print only a count of matching lines per file
      --color[=WHEN]          use markers to highlight the matching strings;
                              WHEN is 'always', 'never', or 'auto'
      --colour[=WHEN]         synonym for --color
  -C, --context NUM           print NUM lines before and after each match
  -e, --regexp PATTERN        use PATTERN for matching (can be repeated)
      --exclude GLOB          skip files that match glob e.g. '*.go'
      --exclude-dir GLOB      skip directories matching glob e.g. 'vendor'
  -F, --fixed-strings         use patterns as strings instead of regular expressions
      --group-separator SEP   separator between context groups (default "--")
  -h, --help                  show this help and exit
  -i, --ignore-case           ignore case distinctions in patterns
      --include GLOB          search only files matching glob e.g. '*.go'
  -l, --files-with-matches    print only names of files with matches
  -n, --line-number           print line number with output lines
      --no-group-separator    do not print separator for matches with context
  -q, --quiet                 suppress all output, exit immediately on first match
  -r, --recursive             search files & directories recursively
  -v, --invert-match          print lines that do not match the pattern
  -w, --word-regexp           matches only whole words

Exit status:
  0 if a match was found, 1 if no match was found, 2 on errors.

Examples:
  needle foo *.go              search Go files for foo
  needle -ri 'TODO|FIXME' src  recursive, case-insensitive search
  needle -C2 needle README.md  show 2 lines of context around matches
`

func printHelp() {
	fmt.Print(helpText)
}
