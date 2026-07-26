package output

import (
	"fmt"
	"os"
	"regexp"

	"github.com/fatih/color"
	"github.com/goziemsunday/needle/internal/search"
	"golang.org/x/term"
)

type Formatter struct {
	Highlight func(a ...any) string
	LineNum   func(a ...any) string
	Sep       func(a ...any) string
}

var (
	Magenta          = color.New(color.FgMagenta).SprintFunc()
	Green            = color.New(color.FgGreen).SprintFunc()
	Red              = color.New(color.FgRed, color.Bold).SprintFunc()
	DefaultFormatter = Formatter{
		Highlight: Red,
		LineNum:   Green,
		Sep:       Magenta,
	}
)

func SetupColors(noColor *bool) {
	if *noColor || !term.IsTerminal(int(os.Stdout.Fd())) {
		color.NoColor = true
	}
}

func FormatMatch(
	m search.Match,
	re *regexp.Regexp,
	f Formatter,
	opts search.Options,
) string {
	highlighted := re.ReplaceAllStringFunc(m.Line, func(s string) string {
		return f.Highlight(s)
	})
	if opts.ShowLineNumbers {
		return fmt.Sprintf("%s%s%s", f.LineNum(m.LineNumber), f.Sep(":"), highlighted)
	}
	return highlighted
}

func GetOutput(r search.Result, opts search.Options, multipleFiles bool) {
	if opts.PrintFilesWithMatches {
		if r.HasMatch {
			fmt.Println(Magenta(r.Path))
		}
	} else if opts.PrintCountPerFile {
		if multipleFiles {
			fmt.Printf("%s%s%d\n", Magenta(r.Path), Magenta(":"), r.Count)
		} else {
			fmt.Println(r.Count)
		}
	} else {
		for _, m := range r.Matches {
			if multipleFiles {
				fmt.Printf("%s%s%s\n", Magenta(r.Path), Magenta(":"), FormatMatch(m, r.RegexpPattern, DefaultFormatter, opts))
			} else {
				fmt.Println(FormatMatch(m, r.RegexpPattern, DefaultFormatter, opts))
			}
		}
	}
}
