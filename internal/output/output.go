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
	Cyan             = color.New(color.FgCyan).SprintFunc()
	DefaultFormatter = Formatter{
		Highlight: Red,
		LineNum:   Green,
		Sep:       Cyan,
	}
)

func SetupColors(when string) {
	switch when {
	case "always":
		color.NoColor = false
	case "never":
		color.NoColor = true
	default: // "auto"
		color.NoColor = !term.IsTerminal(int(os.Stdout.Fd()))
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

func GetOutput(p *matchPrinter, r search.Result, opts search.Options, multipleFiles bool) {
	if opts.Quiet {
		return
	}

	p.Reset(r.Path, multipleFiles)

	if opts.PrintFilesWithMatches {
		if r.HasMatch {
			fmt.Println(Magenta(r.Path))
		}
	} else if opts.PrintCountPerFile {
		if multipleFiles {
			fmt.Printf("%s%s%d\n", Magenta(r.Path), DefaultFormatter.Sep(":"), r.Count)
		} else {
			fmt.Println(r.Count)
		}
	} else {
		for _, m := range r.Matches {
			p.Print(m, r.RegexpPattern)
		}
	}
}

type matchPrinter struct {
	opts          search.Options
	path          string
	multipleFiles bool
	lastPrinted   int
	inGroup       bool // a group was printed in the current file
	prevGroup     bool // a group was printed in an earlier file
}

func NewMatchPrinter(opts search.Options, path string, multipleFiles bool) *matchPrinter {
	return &matchPrinter{opts: opts, path: path, multipleFiles: multipleFiles}
}

// Reset prepares the printer for a new file's results. A group printed in
// any earlier file is remembered so the next file's first group receives
// the separator.
func (p *matchPrinter) Reset(path string, multipleFiles bool) {
	if p.inGroup {
		p.prevGroup = true
	}
	p.path = path
	p.multipleFiles = multipleFiles
	p.inGroup = false
	p.lastPrinted = 0
}

func (p *matchPrinter) Print(m search.Match, re *regexp.Regexp) {
	// separator decision
	contextOn := p.opts.BeforeContext > 0 || p.opts.AfterContext > 0
	if contextOn {
		if !p.inGroup && p.prevGroup {
			// first group of a new file after a group was printed earlier
			fmt.Println(DefaultFormatter.Sep(p.opts.GroupSeparator))
		} else if p.inGroup && m.LineNumber-len(m.Before) > p.lastPrinted+1 {
			// gap within this file
			fmt.Println(DefaultFormatter.Sep(p.opts.GroupSeparator))
		}
	}
	p.inGroup = true

	for _, c := range m.Before {
		if c.Number > p.lastPrinted {
			printContextLine(p.path, c, DefaultFormatter, p.opts, p.multipleFiles)
		}
	}
	printMatchLine(p.path, m, re, DefaultFormatter, p.opts, p.multipleFiles)
	if m.LineNumber > p.lastPrinted {
		p.lastPrinted = m.LineNumber
	}
	for _, c := range m.After {
		if c.Number > p.lastPrinted {
			printContextLine(p.path, c, DefaultFormatter, p.opts, p.multipleFiles)
			p.lastPrinted = c.Number
		}
	}
}

// PrintContextLine prints an after-context line as it streams in
func (p *matchPrinter) PrintContextLine(c search.ContextLine) {
	if c.Number > p.lastPrinted {
		printContextLine(p.path, c, DefaultFormatter, p.opts, p.multipleFiles)
		p.lastPrinted = c.Number
	}
}

// printContextLine prints one context line: "path:num-text" with a
// multi-file prefix, "num-text" when -n is set (dash separator, no
// highlight), or the bare text otherwise
func printContextLine(path string, c search.ContextLine, f Formatter, opts search.Options, multipleFiles bool) {
	if multipleFiles {
		fmt.Printf("%s%s", Magenta(path), f.Sep("-"))
	}
	if opts.ShowLineNumbers {
		fmt.Printf("%s%s%s\n", f.LineNum(c.Number), f.Sep("-"), c.Text)
	} else {
		fmt.Println(c.Text)
	}
}

// printMatchLine prints one match line (highlighted, colon separator)
// with the multi-file path prefix
func printMatchLine(path string, m search.Match, re *regexp.Regexp, f Formatter, opts search.Options, multipleFiles bool) {
	if multipleFiles {
		fmt.Printf("%s%s", Magenta(path), f.Sep(":"))
	}
	fmt.Println(FormatMatch(m, re, f, opts))
}
