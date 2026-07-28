package search

import "regexp"

type Options struct {
	IgnoreCase            bool
	ShowLineNumbers       bool
	PrintCountPerFile     bool
	PrintFilesWithMatches bool
	UseFixedStrings       bool
	RecursiveSearch       bool
	InvertMatch           bool
	Quiet                 bool
	WordBoundary          bool
	Include               string
	Exclude               string
	ExcludeDir            string
}

type Match struct {
	LineNumber int
	Line       string
}

type Result struct {
	Path          string
	Matches       []Match
	Count         int
	HasMatch      bool
	RegexpPattern *regexp.Regexp
	Patterns      []string
}
