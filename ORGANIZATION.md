# Needle - Code Organization Plan

## Current Structure

```
needle/
├── main.go                 # 183 lines — CLI parsing, orchestration, output formatting, exit logic, context setup
└── internal/
    └── search/
        └── search.go       # 354 lines — types, search, file ops, filtering, formatting, concurrent worker pool
```

**Problems:**

- `main.go` mixes CLI setup with presentation logic
- `search.go` has 6+ responsibilities (types, search, file ops, filtering, formatting, concurrency)
- Formatting logic (`Match.Format`, `Formatter`) lives in the search package but is a presentation concern
- Concurrency layer (`workerResult`, `searchFileWorker`, `searchPaths`) is tangled with file operations
- As features are added, both files will balloon past maintainable sizes

## Target Structure

```
needle/
├── main.go                         # Thin entry point (~15 lines)
├── cmd/
│   └── root.go                     # CLI orchestration, flag parsing, mode dispatch (~130 lines)
├── internal/
│   ├── search/
│   │   ├── types.go                # Type definitions (Options, Match, Result) (~30 lines)
│   │   ├── search.go               # Core search logic (Search, SearchStdin, compilePattern) (~80 lines)
│   │   ├── file.go                 # File ops + concurrency (SearchFile, SearchDir, worker pool) (~190 lines)
│   │   └── filter.go               # File filtering (fileMatchesFilters) (~30 lines)
│   └── output/
│       └── output.go               # Output formatting (FormatMatch, GetOutput, Formatter, colors) (~60 lines)
├── ROADMAP.md                      # Feature roadmap
├── ORGANIZATION.md                 # This file
└── justfile                        # Build commands
```

## File Breakdown

### `main.go` (~20 lines)

**Responsibility:** Entry point only. Calls `cmd.Run()` and handles the exit code.

```go
package main

import (
    "os"
    "github.com/goziemsunday/needle/cmd"
)

func main() {
    if err := cmd.Run(); err != nil {
        os.Exit(1)
    }
}
```

### `cmd/root.go` (~130 lines)

**Responsibility:** All CLI orchestration. Migrates everything from current `main.go` except the entry point.

**Contains:**

- `Run() error` — main orchestration function
- Context creation (`context.WithCancel`)
- Flag definitions and parsing
- Mode dispatch (recursive, stdin, file)
- Exit code logic
- Option building

**Does NOT contain:**

- Output formatting (delegates to `output` package)
- Color setup (delegates to `output` package)

### `internal/search/types.go` (~40 lines)

**Responsibility:** Pure type definitions. No logic.

**Contains:**

- `Options` struct
- `Match` struct
- `Result` struct

### `internal/search/search.go` (~80 lines)

**Responsibility:** Core search engine. Pattern compilation and line-by-line scanning.

**Contains:**

- `compilePattern(pattern string, opts Options) (*regexp.Regexp, error)`
- `Search(ctx context.Context, r io.Reader, path, pattern string, opts Options) (Result, error)`
- `SearchStdin(pattern string, opts Options, onMatch func(Match, *regexp.Regexp) bool) (Result, error)`

### `internal/search/file.go` (~190 lines)

**Responsibility:** File system operations and concurrent search orchestration.

**Contains:**

- `SearchFile(ctx context.Context, path, pattern string, opts Options) (Result, error)`
- `SearchDir(ctx context.Context, root, pattern string, opts Options) ([]Result, error)`
- `searchPaths(ctx context.Context, paths []string, pattern string, opts Options) ([]Result, error)` — concurrent orchestrator
- `searchFileWorker(ctx context.Context, jobs <-chan string, workerResults chan<- workerResult, pattern string, opts Options)` — goroutine worker
- `workerResult` struct — carries path, result, and error from workers
- Binary file detection (inline in `SearchFile`)

### `internal/search/filter.go` (~30 lines)

**Responsibility:** File filtering logic.

**Contains:**

- `fileMatchesFilters(name string, opts Options) (bool, error)`

### `internal/output/output.go` (~60 lines)

**Responsibility:** All presentation logic — formatting, colors, output.

**Contains:**

- `Formatter` struct (with `Highlight`, `LineNum`, `Sep` fields)
- `FormatMatch(m Match, re *regexp.Regexp, f Formatter, opts search.Options) string` (was `Match.Format`)
- `GetOutput(r search.Result, opts search.Options, multipleFiles bool)` (was `getOutput`)
- `SetupColors(opts search.Options)` — color initialization logic
- Color variables (`magenta`, `green`, `red`)

## Migration Steps

### Step 1: Create the directory structure

- Create `cmd/` directory
- Create `internal/output/` directory

### Step 2: Create `internal/search/types.go`

- Move `Options`, `Match`, `Result` structs from `search.go`
- Move `Formatter` struct to `output/output.go` instead (it's a presentation type)

### Step 3: Create `internal/output/output.go`

- Move `Formatter` struct from `search.go`
- Move `Match.Format` logic → becomes `FormatMatch` function (takes Match as parameter instead of method)
- Move `getOutput` from `main.go`
- Move color variables and `SetupColors` logic from `main.go`

### Step 4: Split `internal/search/search.go`

- Keep `compilePattern`, `Search`, `SearchStdin` in `search.go`
- Move `SearchFile`, `SearchDir`, `searchPaths`, `searchFileWorker`, `workerResult` to `file.go`
- Move `fileMatchesFilters` to `filter.go`
- Note: `Search` and `SearchFile` now take `context.Context` as first parameter

### Step 5: Create `cmd/root.go`

- Move all CLI logic from `main.go` into `Run() error`
- Create context with cancel: `ctx, cancel := context.WithCancel(context.Background())`
- Pass `ctx` to `search.SearchDir` and `search.SearchFile` calls
- Import and delegate to `output` package for formatting

### Step 6: Slim down `main.go`

- Replace all content with thin entry point calling `cmd.Run()`

### Step 7: Update imports and verify

- Update all import paths
- Run `go build` to verify compilation
- Run `go vet` to check for issues
- Run `go test ./...` to verify no regressions

## Key Decisions

1. **`FormatMatch` becomes a function, not a method** — Since `Formatter` and `Match` are now in different packages, the formatting logic lives in the `output` package as a standalone function that takes both as parameters.

2. **`Formatter` moves to `output` package** — It's a presentation type, not a search type. The search package should not know about colors.

3. **`workerResult` stays in `file.go`** — It's an internal type used only by the worker pool, not exported. No need to put it in `types.go`.

4. **`cmd.Run()` returns `error`** — The main function handles the exit code based on the error, keeping `cmd` package testable.

5. **No circular imports** — `cmd` imports `search` and `output`. `output` imports `search` (for types). `search` imports nothing from the project. Clean dependency graph.

6. **Context flows through `cmd` → `search/file.go` → `search/search.go`** — The context is created in `cmd/root.go` and threaded through to `SearchFile` and `Search`. Workers check `ctx.Done()` for cancellation. This enables future features like timeout and early termination.
