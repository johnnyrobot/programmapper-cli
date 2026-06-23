# Program Pathways Mapper CLI

**Every California Community College's program maps - degrees, courses, transfer paths, and careers - in one scriptable CLI with an offline SQLite catalog and term-by-term planning no other ProgramMapper tool has.**

Program Pathways Mapper publishes the official 'how to finish your degree' roadmaps for hundreds of California community colleges, but only through a click-heavy, one-college-at-a-time web app with no API key and no export. This CLI mirrors a college's full catalog into local SQLite, then lets you - or an AI advisor - search across colleges, build a term-by-term plan with units rollups, compare two programs, find every program that requires a course, and diff catalog years. Offline, scriptable, and agent-native.

## Install

The recommended path installs both the `programmapper-cli` binary and the `pp-programmapper` agent skill (Claude Code, Codex, Cursor, Gemini CLI, GitHub Copilot, and other agents supported by the upstream [`skills`](https://github.com/vercel-labs/skills) CLI) in one shot:

```bash
npx -y @mvanhorn/printing-press-library install programmapper
```

For CLI only (no skill):

```bash
npx -y @mvanhorn/printing-press-library install programmapper --cli-only
```

For skill only — installs the skill into the same agents as the default command above, but skips the CLI binary (use this to update or reinstall just the skill):

```bash
npx -y @mvanhorn/printing-press-library install programmapper --skill-only
```

To constrain the skill install to one or more specific agents (repeatable — agent names match the [`skills`](https://github.com/vercel-labs/skills) CLI):

```bash
npx -y @mvanhorn/printing-press-library install programmapper --agent claude-code
npx -y @mvanhorn/printing-press-library install programmapper --agent claude-code --agent codex
```

### Without Node (Go fallback)

If `npx` isn't available (no Node, offline), install the CLI directly via Go (requires Go 1.26.4 or newer):

```bash
go install github.com/mvanhorn/printing-press-library/library/other/programmapper/cmd/programmapper-cli@latest
```

This installs the CLI only — no skill.

### Pre-built binary

Download a pre-built binary for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/programmapper-current). On macOS, clear the Gatekeeper quarantine: `xattr -d com.apple.quarantine <binary>`. On Unix, mark it executable: `chmod +x <binary>`.

<!-- pp-hermes-install-anchor -->
## Install for Hermes

Install the CLI binary first. The installer writes binaries to a per-user managed bin directory by default: `$HOME/.local/bin` on macOS/Linux and `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows.

```bash
npx -y @mvanhorn/printing-press-library install programmapper --cli-only
```

Then install the focused Hermes skill.

From the Hermes CLI:

```bash
hermes skills install mvanhorn/printing-press-library/cli-skills/pp-programmapper --force
```

Inside a Hermes chat session:

```bash
/skills install mvanhorn/printing-press-library/cli-skills/pp-programmapper --force
```

Restart the Hermes session or gateway if the newly installed skill is not visible immediately.

## Install for OpenClaw
Install both the CLI binary and the focused OpenClaw skill. The installer defaults binaries to a per-user bin directory (`$HOME/.local/bin` on macOS/Linux, `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows):

```bash
npx -y @mvanhorn/printing-press-library install programmapper --agent openclaw
```

Restart the OpenClaw session or gateway if the newly installed skill is not visible immediately.

## Use with Claude Desktop

This CLI ships an [MCPB](https://github.com/modelcontextprotocol/mcpb) bundle — Claude Desktop's standard format for one-click MCP extension installs (no JSON config required).

To install:

1. Download the `.mcpb` for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/programmapper-current).
2. Double-click the `.mcpb` file. Claude Desktop opens and walks you through the install.

Requires Claude Desktop 1.0.0 or later. Pre-built bundles ship for macOS Apple Silicon (`darwin-arm64`) and Windows (`amd64`, `arm64`); for other platforms, use the manual config below.

<details>
<summary>Manual JSON config (advanced)</summary>

If you can't use the MCPB bundle (older Claude Desktop, unsupported platform), install the MCP binary and configure it manually.


```bash
go install github.com/mvanhorn/printing-press-library/library/other/programmapper/cmd/programmapper-mcp@latest
```

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "programmapper": {
      "command": "programmapper-mcp"
    }
  }
}
```

</details>

## Quick Start

```bash
# Confirm the CLI is wired up - no API key needed (the data is public).
programmapper-cli doctor --dry-run

# Resolve a college to its active site-content id.
programmapper-cli colleges resolve --vanity-url https://la-mission.programmapper.ws

# Download the college's full catalog (programs, maps, courses) into local SQLite.
programmapper-cli mirror la_mission

# Search the mirrored catalog offline for matching programs.
programmapper-cli search nursing --type programs --data-source local --limit 10

# Build the term-by-term plan (program id from the search results).
programmapper-cli plan 4a0cb2c2-b22f-324d-834e-80cbc2bde5f4

```

## Unique Features

These capabilities aren't available in any other tool for this API.

### Plan & compare from a local catalog
- **`plan`** — Build a program's full semester-by-semester plan with per-term and cumulative units, expanding required courses and 'choose one of' options.

  _When an agent is asked to build a 2-year academic plan, this returns the whole term structure with units in one structured call instead of paging a UI._

  ```bash
  programmapper-cli plan 4a0cb2c2-b22f-324d-834e-80cbc2bde5f4 --agent
  ```
- **`compare`** — Show two programs side by side: shared courses, unique courses, and per-program units totals.

  _Lets a student or agent decide between two programs by the actual course overlap and unit cost, not prose._

  ```bash
  programmapper-cli compare 4a0cb2c2-b22f-324d-834e-80cbc2bde5f4 7b7428c7-ab0e-4e00-b9eb-08091c31b34c --json
  ```
- **`diff-years`** — Diff a program's current map against a prior catalog year (2019-2025) and report added, removed, and changed courses plus unit deltas.

  _Answers 'did my program's requirements change since the catalog year I started under?' without manual side-by-side reading._

  ```bash
  programmapper-cli diff-years 4a0cb2c2-b22f-324d-834e-80cbc2bde5f4 --from-year 2023 --json
  ```

### Reverse-lookups the website can't do
- **`course-programs`** — List every program and map at the synced college whose term plan includes a given course.

  _Answers a counselor's 'which programs require this course?' in one call instead of opening every map._

  ```bash
  programmapper-cli course-programs "NURSING 090" --json
  ```
- **`bottlenecks`** — Rank courses across the synced college by how many program maps require them and by units, surfacing high-leverage and high-load courses.

  _Shows advisors which courses unlock the most programs so scheduling and tutoring effort goes where it matters._

  ```bash
  programmapper-cli bottlenecks --limit 20 --json
  ```

### Cross-college & transfer
- **`search`** — Full-text search programs and courses across every synced college at once, offline.

  _Finds a program or course by keyword across many colleges in one query instead of visiting each college's site._

  ```bash
  programmapper-cli search nursing --type programs --data-source local --limit 10
  ```
- **`transfer-options`** — Combine a program's CSU/UC linked transfer colleges and linked program maps into one view.

  _Gives an agent the full set of transfer destinations for a program in one structured response._

  ```bash
  programmapper-cli transfer-options 4a0cb2c2-b22f-324d-834e-80cbc2bde5f4 --json
  ```

## Recipes


### Mirror a college, then plan a degree

```bash
programmapper-cli mirror la_mission --maps
```

Cache L.A. Mission's full catalog (programs, maps, courses) locally so plan, compare, course-programs, and bottlenecks run offline. Run this once before the other commands.

### Narrow a deeply nested plan for an agent

```bash
programmapper-cli plan 4a0cb2c2-b22f-324d-834e-80cbc2bde5f4 --agent --select terms.term_number,terms.items.code,terms.items.name,terms.items.min_units
```

Program maps are deeply nested; --select pulls only the term, course title, and units so an agent isn't flooded with the full map payload.

### Find which programs require a course

```bash
programmapper-cli course-programs "NURSING 090" --json
```

Reverse-lookup every program at the mirrored college whose plan includes the course - impossible in the web UI.

### See what changed since your start year

```bash
programmapper-cli diff-years 4a0cb2c2-b22f-324d-834e-80cbc2bde5f4 --from-year 2023
```

Diff the current Vocational Nursing map against the 2023 catalog to spot added, removed, or changed courses.

### Compare two programs head to head

```bash
programmapper-cli compare 4a0cb2c2-b22f-324d-834e-80cbc2bde5f4 7b7428c7-ab0e-4e00-b9eb-08091c31b34c --json
```

Show shared vs. unique courses and total units for two programs side by side.

## Usage

Run `programmapper-cli --help` for the full command reference and flag list.

## Paths & environment variables

This CLI separates local files into four path kinds:

| Kind | Contents |
|------|----------|
| `config` | User-editable settings such as `config.toml` and saved profiles |
| `data` | Durable local data such as `data.db` |
| `state` | Runtime state such as persisted queries, jobs, and `teach.log` |
| `cache` | Regenerable HTTP/cache files |

Each kind resolves independently. The ladder is:

1. Per-kind env var: `PROGRAMMAPPER_CONFIG_DIR`, `PROGRAMMAPPER_DATA_DIR`, `PROGRAMMAPPER_STATE_DIR`, or `PROGRAMMAPPER_CACHE_DIR`
2. `--home <dir>` for this invocation
3. `PROGRAMMAPPER_HOME` for a flat relocated root
4. XDG env vars: `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, `XDG_CACHE_HOME`
5. Platform defaults matching existing installs

For containers and agent sandboxes, prefer a single relocated root:

```bash
export PROGRAMMAPPER_HOME=/srv/programmapper
programmapper-cli doctor
```

Under `PROGRAMMAPPER_HOME=/srv/programmapper`, the four dirs resolve to `/srv/programmapper/config`, `/srv/programmapper/data`, `/srv/programmapper/state`, and `/srv/programmapper/cache`.

MCP servers do not receive CLI flags from the host. Put relocation in the host `env` block:

```json
{
  "mcpServers": {
    "programmapper": {
      "command": "programmapper-mcp",
      "env": {
        "PROGRAMMAPPER_HOME": "/srv/programmapper"
      }
    }
  }
}
```

Precedence matters in fleets: an ambient per-kind variable such as `PROGRAMMAPPER_DATA_DIR` overrides an explicit `--home` for that kind. Use `PROGRAMMAPPER_HOME` or the per-kind variables for durable fleet relocation; treat `--home` as the weaker per-invocation lever.

Relocation is one-way. Unsetting `PROGRAMMAPPER_HOME` does not move files back to platform defaults, and `doctor` cannot find files left under a former root. Move the files manually before unsetting relocation variables.

Existing installs keep working because the platform-default rung matches the legacy layout. Run `programmapper-cli doctor --fail-on warn` to check path warnings in automation.

## Commands

### colleges

California Community Colleges in the ProgramMapper registry

- **`programmapper-cli colleges get`** - Get one college by its registry id (e.g. la_mission)
- **`programmapper-cli colleges list`** - List every college in the registry (id, active site-content id, prior-year ids)
- **`programmapper-cli colleges resolve`** - Resolve the college(s) for a ProgramMapper vanity URL

### courses

Individual courses referenced by program maps

- **`programmapper-cli courses get`** - Get a course's detail (units, description, requisites)
- **`programmapper-cli courses high-schools`** - List high schools offering this course for dual enrollment

### interest-clusters

Guided Pathways 'Areas of Interest' grouping programs (program-groups)

- **`programmapper-cli interest-clusters <site_content_id> <cluster_id>`** - Get an interest cluster (program group) and the programs grouped under it

### meta

API version and health

- **`programmapper-cli meta`** - Get the ProgramMapper API version

### occupations

Career and salary outcomes keyed by SOC code

- **`programmapper-cli occupations`** - Batch-fetch career data (salary, job growth) for SOC codes

### program-maps

Semester-by-semester course plans for a program pathway

- **`programmapper-cli program-maps get`** - Get a program map: terms -> course/choice opportunities with units
- **`programmapper-cli program-maps linked`** - Get a linked (transfer-partner) program map

### programs

Degrees and certificates offered at a college

- **`programmapper-cli programs get`** - Get a program's full detail (description, award, pathways, map refs)
- **`programmapper-cli programs linked`** - Get a linked (transfer-partner) program
- **`programmapper-cli programs newer-year-link`** - Get link data to the same program in a newer catalog year
- **`programmapper-cli programs pathways`** - List a program's pathway summaries (each pathway is an alternate route)
- **`programmapper-cli programs prior-years`** - List prior catalog years (2019+) that still have a published map for this program
- **`programmapper-cli programs search`** - Full-text search programs by title and description

### site-contents

Per-college, per-year catalog config (one site-content per college-year)

- **`programmapper-cli site-contents get`** - Get a college-year's config (name, year, theme, titles)
- **`programmapper-cli site-contents home`** - Get the home page: interest clusters (Areas of Interest), title, about

### transfer

CSU/UC transfer pathways and 'choose one of' opportunities

- **`programmapper-cli transfer choice-opportunity`** - Get a 'choose one of' opportunity (idealized program-pathway choice) detail
- **`programmapper-cli transfer linked-college`** - Get a CSU/UC transfer college's detail for a transfer pathway


## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
programmapper-cli colleges list

# JSON for scripting and agents
programmapper-cli colleges list --json

# Filter to specific fields
programmapper-cli colleges list --json --select id,name,status

# Dry run — show the request without sending
programmapper-cli colleges list --dry-run

# Agent mode — JSON + compact + no prompts in one flag
programmapper-cli colleges list --agent
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select id,name` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Explicit retries** - add `--idempotent` to create retries when a no-op success is acceptable
- **Confirmable** - `--yes` for explicit confirmation of destructive actions
- **Piped input** - write commands can accept structured input when their help lists `--stdin`
- **Offline-friendly** - sync/search commands can use the local SQLite store when available
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `5` API error, `7` rate limited, `10` config error.

## Health Check

```bash
programmapper-cli doctor
```

Verifies configuration and connectivity to the API.

## Configuration

Run `programmapper-cli doctor` to see the resolved config, data, state, and cache directories. The platform-default config path is `~/.config/programmapper-cli/config.toml`; `--home`, `PROGRAMMAPPER_HOME`, and per-kind env vars can relocate it.

Static request headers can be configured under `headers`; per-command header overrides take precedence.

## Troubleshooting
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

### API-specific
- **Commands return 'no local mirror' or empty results** — Run 'programmapper-cli mirror <college>' first to populate the local SQLite catalog.
- **HTTP 403 / rate-limited from the API** — The ProgramMapper WAF throttles bursts per IP; the CLI uses a browser-fingerprinted client and paces requests - retry after a short pause, or lower concurrency.
- **'colleges resolve' returns an empty list** — Pass the full URL including scheme, e.g. https://la-mission.programmapper.ws (not just the hostname).

## HTTP Transport

This CLI uses Chrome-compatible HTTP transport for browser-facing endpoints. It does not require a resident browser process for normal API calls.

---

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
