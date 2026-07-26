package search

import "path/filepath"

func fileMatchesFilters(name string, opts Options) (bool, error) {
	// if --include is set, skip files that don't match the glob
	if opts.Include != "" {
		matched, err := filepath.Match(opts.Include, name)
		if err != nil {
			return false, err
		}
		if !matched {
			return false, nil
		}
	}

	// if --exclude is set, skip files that match the glob
	if opts.Exclude != "" {
		matched, err := filepath.Match(opts.Exclude, name)
		if err != nil {
			return false, err
		}
		if matched {
			return false, nil
		}
	}

	return true, nil
}
