# Needle

Grep-like search tool in Go.

## Commands

- `just build` — builds to `bin/needle`
- `just run -- <args>` — runs via `go run`
- `just test` — runs `go test ./...`
- `go vet ./...` — static analysis (no linter configured)
- No formatter configured; use `gofmt` manually

## Project Layout

- `main.go` — CLI entry point, flag parsing, output formatting, orchestration
- `internal/search/search.go` — core search engine, file ops, concurrent worker pool, filtering
- `ORGANIZATION.md` — planned refactoring (not yet implemented)
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
- `main.go` contains formatting logic that ORGANIZATION.md plans to move to `internal/output/`
- Go 1.26.2 required (check `go.mod`)
