# Needle - Code Organization

## Current Structure

```
needle/
├── main.go                         # 13 lines — thin entry point, calls cmd.Run()
├── cmd/
│   └── root.go                     # CLI orchestration, flag parsing, mode dispatch
├── internal/
│   ├── search/
│   │   ├── types.go                # Type definitions (Options, Match, Result)
│   │   ├── search.go               # Core search (Search, SearchStdin, compilePattern)
│   │   ├── file.go                 # File ops + concurrency (SearchFile, SearchDir, worker pool)
│   │   └── filter.go               # File filtering (fileMatchesFilters)
│   └── output/
│       └── output.go               # Output formatting (FormatMatch, GetOutput, Formatter, colors)
├── ROADMAP.md                      # Feature roadmap
├── ORGANIZATION.md                 # This file
└── justfile                        # Build commands
```

## Dependency Graph

```
main → cmd → search
           → output → search
search → (stdlib only)
```

No circular imports.

## File Responsibilities

### `main.go` (~13 lines)

Entry point only. Calls `cmd.Run()` and exits with code 1 on error.

### `cmd/root.go` (~149 lines)

All CLI orchestration. Contains:

- `Run() error` — main orchestration function
- Context creation (`context.WithCancel`)
- Flag definitions and parsing
- Mode dispatch (recursive, stdin, file)
- Exit code logic (returns error on no match)
- Option building

Does NOT contain output formatting or color setup — delegates to `output` package.

### `internal/search/types.go` (~28 lines)

Pure type definitions: `Options`, `Match`, `Result`. No logic.

### `internal/search/search.go` (~102 lines)

Core search engine. Pattern compilation and line-by-line scanning.

- `compilePattern(pattern string, opts Options) (*regexp.Regexp, error)`
- `Search(ctx context.Context, r io.Reader, path, pattern string, opts Options) (Result, error)`
- `SearchStdin(ctx context.Context, pattern string, opts Options, onMatch func(Match, *regexp.Regexp) bool) (Result, error)`

### `internal/search/file.go` (~199 lines)

File system operations and concurrent search orchestration.

- `SearchFile(ctx context.Context, path, pattern string, opts Options) (Result, error)`
- `SearchDir(ctx context.Context, root, pattern string, opts Options) ([]Result, error)`
- `searchPaths(ctx context.Context, paths []string, pattern string, opts Options) ([]Result, error)` — concurrent orchestrator
- `searchFileWorker(ctx context.Context, jobs <-chan string, workerResults chan<- workerResult, pattern string, opts Options)` — goroutine worker
- `workerResult` struct — carries path, result, and error from workers
- Binary file detection (inline in `SearchFile`)

### `internal/search/filter.go` (~29 lines)

File filtering logic.

- `fileMatchesFilters(name string, opts Options) (bool, error)`

### `internal/output/output.go` (~71 lines)

All presentation logic — formatting, colors, output.

- `Formatter` struct (with `Highlight`, `LineNum`, `Sep` fields)
- `DefaultFormatter` — pre-configured Formatter with color functions
- `FormatMatch(m search.Match, re *regexp.Regexp, f Formatter, opts search.Options) string`
- `GetOutput(r search.Result, opts search.Options, multipleFiles bool)`
- `SetupColors(noColor *bool)` — color initialization
- Exported color functions: `Magenta`, `Green`, `Red`

## Known Issues

- `SearchStdin` accepts `ctx context.Context` but does not check `ctx.Done()` — it could block on `scanner.Scan()` indefinitely if the context is cancelled while reading stdin.
