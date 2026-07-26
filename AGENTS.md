# Needle

Grep-like search tool in Go.

## Commands

- `just build` — builds to `bin/needle`
- `just run -- <args>` — runs via `go run`
- `just test` — runs `go test ./...`
- `go vet ./...` — static analysis (no linter configured)
- No formatter configured; use `gofmt` manually

## Project Layout

- `main.go` — thin entry point, calls `cmd.Run()`
- `cmd/root.go` — CLI orchestration, flag parsing, mode dispatch, output formatting
- `internal/search/` — core search engine (`search.go`), file ops + workers (`file.go`), filtering (`filter.go`), types (`types.go`)
- `internal/output/output.go` — output formatting, colors, `FormatMatch`, `GetOutput`
- `ORGANIZATION.md` — architecture documentation
- `ROADMAP.md` — feature roadmap with implementation notes

## Conventions

- CLI flags use `pflag` (POSIX-style: `-i`, `--ignore-case`)
- Exit code 1 when no matches found, 0 on match
- Hidden directories (names starting with `.`) are skipped in recursive mode
- Binary files silently skipped (null byte in first 512 bytes)
- `--include` / `--exclude` use `filepath.Match` glob syntax
- `--exclude-dir` also uses glob syntax

## Gotchas

- No tests exist yet; `just test` will pass vacuously
- Go 1.26.2 required (check `go.mod`)
- `SearchStdin` accepts a `ctx context.Context` but never checks `ctx.Done()` — it can block on `scanner.Scan()` if context is cancelled while reading stdin
