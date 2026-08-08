# AGENTS.md

`programmapper-cli` is a Go 1.26 command-line tool and companion MCP server that mirror California Community College "Program Pathways Mapper" catalogs into local SQLite and expose scriptable, agent-native commands. Two binaries build from `cmd/`: `programmapper-cli` and `programmapper-mcp`. Business logic lives in `internal/` (`cli`, `client`, `store`, `cache`, `config`, `cliutil`, `types`, `mcp`).

> **This directory is generated output** from [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press). Treat systemic bugs as upstream Printing Press fixes first. If you must edit the generated tree, keep it narrow and record the intent under `.printing-press-patches/` (a durable reprint-guard) — a fresh print overwrites unrecorded hand-edits. Do not hand-edit the release ledger (`CHANGELOG.md`, `.printing-press-release.json`, `var version = …`); the `mvanhorn/printing-press-library` publish workflow stamps the `YYYY.M.N` version on merge.

## Setup
```bash
go install github.com/johnnyrobot/programmapper-cli/cmd/programmapper-cli@latest
```
Or work from a checkout: `go build ./cmd/...`. Requires Go 1.26+. No API key is needed — the catalog data is public.

## Build & Run
```bash
make build            # bin/programmapper-cli
make build-mcp        # bin/programmapper-mcp (MCP server for Claude Desktop, etc.)
make build-all        # both
make install          # go install ./cmd/programmapper-cli
```
Typical run: `programmapper-cli mirror la_mission` to populate the local SQLite catalog, then offline commands like `plan`, `compare`, `course-programs`, `bottlenecks`, `search --data-source local`. Start any investigation with `programmapper-cli doctor --json` and `programmapper-cli agent-context --pretty` for runtime truth; use `programmapper-cli <command> --help` and `--dry-run` before mutating/network calls.

## Testing
```bash
make test             # go test ./...
```
Unit tests live beside the code they cover in `internal/cli/*_test.go` (e.g. `bottlenecks_test.go`, `compare_test.go`). Before considering a change done: `make test` and `make lint` both pass, and the CLI still builds (`make build-all`).

## Code Style
- Go 1.26; formatted with `gofmt` + `goimports` (enabled as golangci formatters).
- `make lint` runs golangci-lint with `errorlint, govet, ineffassign, staticcheck, unused, bodyclose, noctx, rowserrcheck, sqlclosecheck` (`.golangci.yml`). Fix findings rather than suppressing them.
- One cobra command per file in `internal/cli`; keep the CLI non-interactive (flags only), print `--json` to stdout and errors to stderr, and honor the shared `--agent` / `--dry-run` / `--yes --no-input` conventions.
- SQLite access goes through `internal/store` using `modernc.org/sqlite` (pure Go — no cgo); close rows/statements (the linters enforce this).

## Commit & PR Conventions
- Branch `main`; concise imperative commit subjects (e.g. "Make module go-install-able", "Add HANDOFF.md").
- Do not commit generated/regenerable artifacts: `*.db*`, `/build/`, and the compiled `/programmapper-cli` / `/programmapper-mcp` binaries are git-ignored — keep them out.
- Releases are tag-driven: `git tag vX.Y.Z && git push origin vX.Y.Z` triggers GoReleaser (`.github/workflows/release.yml`).

## Security & Data
- No secrets in this repo — the ProgramMapper API needs no key. Don't introduce committed credentials.
- The API sits behind a WAF that rate-limits/403s bursts per IP; the `client` package uses a Chrome-fingerprinted transport and paces requests. Retry after a pause instead of raising concurrency.
- Mirrored catalogs are written to the local `data` dir (`data.db`); relocate with `PROGRAMMAPPER_HOME` or the per-kind `PROGRAMMAPPER_{CONFIG,DATA,STATE,CACHE}_DIR` env vars for sandboxes/fleets. Relocation is one-way (unsetting does not move files back).
