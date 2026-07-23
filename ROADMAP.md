# Needle - Feature Roadmap

## Features

### 1. `-v` Invert Match

**Priority:** High | **Difficulty:** Easy

Inverts the match logic — prints lines that **don't** match the pattern.

**Implementation:**

- Add `Invert bool` field to `Options` struct in `internal/search/types.go`
- Add `--invert-match` / `-v` flag in `cmd/root.go`
- In `Search()`, flip the condition: `if match != opts.Invert` instead of `if match`
- Same logic applies to `SearchStdin()` callback

**Lines affected:** ~10 lines across `search.go` and `cmd/root.go`

---

### 2. `-q` Quiet Mode

**Priority:** High | **Difficulty:** Medium

Exits immediately on first match without printing anything. Returns exit code 0 if found, 1 if not.

**Implementation:**

- Add `Quiet bool` field to `Options`
- Add `--quiet` / `-q` flag in `cmd/root.go`
- In all search functions, return early after first match when quiet is set
- In `cmd/root.go`, set exit code based on whether any match was found
- **Concurrency note:** With the worker pool, quiet mode should cancel the context on first match. The worker that finds the first match calls `cancel()`, and all other workers stop via `ctx.Done()` checks already in place. Ensure only one result is returned by using `sync.Once` or a channel-based signal.

**Lines affected:** ~20-25 lines

---

### 3. `--help` Output

**Priority:** High | **Difficulty:** Easy

Prints a usage message with descriptions and examples when `-h` or `--help` is passed, or when no arguments are provided.

**Implementation:**

- Create a `printHelp()` function in `cmd/root.go`
- Include: usage syntax, flag descriptions, 2-3 practical examples
- Call it when `len(os.Args) < 2` or when help flag is set
- Use `pflag`'s built-in help or write a custom one for more control

**Lines affected:** ~30-40 lines

---

### 4. `-w` Word Boundaries

**Priority:** Medium | **Difficulty:** Easy

Matches only whole words, not substrings. Prevents "cat" from matching inside "concatenate".

**Implementation:**

- Add `WordBoundary bool` field to `Options`
- Add `--word-regexp` / `-w` flag in `cmd/root.go`
- In `compilePattern()`, wrap the final regex pattern with `\b` assertions when enabled
- Pattern becomes: ``fmt.Sprintf(`%s\b%s\b`, prefix, pattern)`` (simplified)

**Lines affected:** ~5-10 lines in `compilePattern()`

---

### 5. Tests

**Priority:** High | **Difficulty:** Medium

Table-driven unit tests for the search engine.

**Implementation:**

- Create `internal/search/search_test.go` for core search tests
- Create `internal/search/file_test.go` for file and concurrency tests
- Test cases to cover:
  - Basic regex match
  - Fixed-string match (`-F`)
  - Case-insensitive (`-i`)
  - Inverted match (`-v`)
  - Word boundaries (`-w`)
  - Binary file detection/skipping
  - Recursive directory search
  - stdin search
  - **Concurrent search with multiple files (worker pool)**
  - **Context cancellation mid-search**
- Use `t.TempDir()` for creating test files
- Use `testing.T` table-driven test pattern
- Run tests with `-race` flag to detect race conditions
- Add benchmarks for future performance work

**Lines affected:** ~150-200 lines (new test files)

---

### 6. `-e` Multiple Patterns

**Priority:** Medium | **Difficulty:** Medium

Allows searching for multiple patterns in a single call.

**Implementation:**

- Change pattern handling from single `string` to `[]string`
- Add `--regexp` / `-e` flag that can be specified multiple times (use `pflag.StringArrayVar`)
- In `compilePattern()`, compile each pattern into a separate `regexp.Regexp`
- Store compiled patterns as `[]*regexp.Regexp` in `Result`
- In `Search()` and matching logic, check each pattern — match if any pattern matches
- Update `cmd/root.go` to accept both positional patterns and `-e` flags, combining them

**Lines affected:** ~30-40 lines across `search.go` and `cmd/root.go`

---

### 7. `-A`/`-B`/`-C` Context Lines

**Priority:** Medium | **Difficulty:** Hard

Shows lines before and/or after each match.

**Implementation:**

- Add `AfterContext int`, `BeforeContext int`, `Context int` fields to `Options`
- Add `-A`, `-B`, `-C` flags in `cmd/root.go`
- `-C N` sets both `AfterContext` and `BeforeContext` to N
- Use a ring buffer or slice to hold the last N lines for `-B` lookback
- In `Search()`:
  - Maintain a buffer of recent lines
  - On match: flush `-B` buffer, print match, then print next `-A` lines
  - Handle overlapping context regions (don't double-print)
- Add `--group-separator` (optional, default `--`) to visually separate context groups
- **Concurrency note:** Context lines are per-file, which is correct since each worker processes files independently. The buffer lives inside `Search()` and is scoped to a single file read.

**Lines affected:** ~60-80 lines

---

### 8. Config File (`~/.needlerc`)

**Priority:** Low | **Difficulty:** Hard

Stores default preferences so users don't repeat flags every time.

**Implementation:**

- Add TOML dependency (`github.com/BurntSushi/toml` or similar)
- Define config struct matching the `Options` fields
- On startup, check for `~/.needlerc` using `os.UserHomeDir()`
- If found, parse TOML and merge with flag values (flags override config)
- Config file takes lowest priority: defaults → config file → CLI flags
- Skip silently if config file doesn't exist
- Add `--no-config` flag to ignore the config file

**Lines affected:** ~80-100 lines plus new dependency

---

## Implementation Order

1. `-v` invert match
2. `-q` quiet mode
3. `--help`
4. `-w` word boundaries
5. Tests
6. `-e` multiple patterns
7. `-A`/`-B`/`-C` context lines
8. Config file
