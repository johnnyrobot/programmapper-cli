# CLAUDE.md

`programmapper-cli` is a Go CLI + MCP server that mirrors California Community College "Program Pathways Mapper" catalogs (degrees, courses, transfer paths, careers) into local SQLite and exposes scriptable, agent-native commands (search, plan, compare, diff-years, bottlenecks) that the click-heavy web app can't. The catalog data is public — no API key.

> This tree is **generated output** from [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press). Prefer fixing systemic issues upstream in Printing Press; keep any local edit narrow and recorded (see Gotchas).

## Architecture
- **Entry points:** `cmd/programmapper-cli/main.go` (the CLI) and `cmd/programmapper-mcp/main.go` (the MCP server, published binary `programmapper-mcp`). Both are thin mains over `internal/`.
- **`internal/` packages:**
  - `cli/` — cobra commands, one file per command (`colleges_*.go`, `plan`, `compare`, `bottlenecks`, `search`, `mirror`, `doctor`, `agent_context.go`, `which`, …) plus `*_test.go`.
  - `client/` — HTTP client for the ProgramMapper API. Uses a **Chrome-fingerprinted transport** (`github.com/enetx/surf` / `enetx/http`) to get past the WAF; not a resident browser.
  - `store/` — local SQLite catalog via `modernc.org/sqlite` (pure-Go, no cgo). Populated by `mirror`.
  - `cache/`, `config/`, `cliutil/`, `types/`, `mcp/` — HTTP cache, path/config resolution, shared CLI helpers, shared types, MCP wiring.
- **Data flow:** `mirror <college>` fetches programs/maps/courses into `data.db`; offline commands (`plan`, `compare`, `course-programs`, `bottlenecks`, `search --data-source local`) read from that SQLite store.
- **Path kinds:** config / data / state / cache each resolve independently (per-kind env var → `--home` → `PROGRAMMAPPER_HOME` → XDG → platform default). See README "Paths & environment variables".

## Commands
```bash
make build          # go build -o bin/programmapper-cli ./cmd/programmapper-cli
make build-mcp      # go build -o bin/programmapper-mcp ./cmd/programmapper-mcp
make build-all      # both binaries
make test           # go test ./...
make lint           # golangci-lint run
make install        # go install ./cmd/programmapper-cli
# Private module: authenticate git + set GOPRIVATE before `go install`/`go get`.
export GOPRIVATE=github.com/johnnyrobot
# Runtime discovery (prefer over a memorized command list):
./bin/programmapper-cli doctor --json
./bin/programmapper-cli agent-context --pretty
./bin/programmapper-cli which "<capability>" --json
./bin/programmapper-cli <command> --help
```

## Conventions
- **Agent-first CLI:** non-interactive, every input is a flag, `--json` to stdout / errors to stderr. `--agent` bundles JSON + compact + no color + non-interactive. Use `--dry-run` before anything that hits the network; `--yes --no-input` only once target and side effects are clear.
- **Exit codes:** `0` ok, `2` usage, `3` not found, `5` API error, `7` rate limited, `10` config error.
- **Lint:** golangci-lint with `errorlint, govet, staticcheck, unused, bodyclose, noctx, rowserrcheck, sqlclosecheck`; formatters `gofmt` + `goimports` (see `.golangci.yml`). Run `make lint` before finishing.
- Go 1.26. One cobra command per file in `internal/cli`; add a matching `*_test.go`.

## Gotchas & Constraints
- **Generated tree.** A fresh print can overwrite this whole directory. Ad-hoc hand-edits do **not** survive on their own — record every intentional change under `.printing-press-patches/` (parallel to `.printing-press.json`) so a regen carries it forward.
- **Do not hand-bump the release ledger.** `CHANGELOG.md`, `.printing-press-release.json`, and `var version = …` are stamped by the `mvanhorn/printing-press-library` publish workflow (final `YYYY.M.N` assigned on merge). Preserve those files on reprint; never edit them for bookkeeping.
- **API is WAF-guarded.** Bursts get HTTP 403 / rate-limited per IP; the client paces requests — retry after a pause, don't crank concurrency.
- **Offline commands need a mirror first.** "no local mirror" / empty results ⇒ run `mirror <college>`.
- **`colleges resolve` needs the full URL** including scheme (`https://la-mission.programmapper.ws`).
- `*.db`, `*.db-shm`, `*.db-wal`, `/build/`, and the built binaries are git-ignored (regenerable) — don't commit them.
- Releases are cut by pushing a `v*` tag (GoReleaser, `.github/workflows/release.yml`); `.goreleaser.yaml` is the config.
