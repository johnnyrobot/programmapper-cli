---
name: pp-programmapper
description: "Every California Community College's program maps - degrees, courses, transfer paths, and careers - in one scriptable CLI with an offline SQLite catalog and term-by-term planning no other ProgramMapper tool has. Trigger phrases: `map out a degree at a community college`, `what courses do I need for nursing at LA Mission`, `compare two community college programs`, `which programs require this course`, `plan my CSU transfer pathway`, `use programmapper`, `run programmapper`."
author: "johnnyrobot"
license: "Apache-2.0"
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - programmapper-pp-cli
    install:
      - kind: go
        bins: [programmapper-pp-cli]
        module: github.com/mvanhorn/printing-press-library/library/other/programmapper/cmd/programmapper-pp-cli
---

# Program Pathways Mapper — Printing Press CLI

## Prerequisites: Install the CLI

This skill drives the `programmapper-pp-cli` binary. **You must verify the CLI is installed before invoking any command from this skill.** If it is missing, install it first:

1. Install via the Printing Press installer. It defaults binaries to `$HOME/.local/bin` on macOS/Linux and `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows:
   ```bash
   npx -y @mvanhorn/printing-press-library install programmapper --cli-only
   ```
2. Verify: `programmapper-pp-cli --version`
3. Ensure the reported install directory is on `$PATH` for the agent/runtime that will invoke this skill.

If the `npx` install fails (no Node, offline, etc.), fall back to a direct Go install (requires Go 1.26.4 or newer). This installs into `$GOPATH/bin` (default `$HOME/go/bin`), so add that directory to `$PATH` instead:

```bash
go install github.com/mvanhorn/printing-press-library/library/other/programmapper/cmd/programmapper-pp-cli@latest
```

If `--version` reports "command not found" after install, the runtime cannot see the binary directory on `$PATH`. Do not proceed with skill commands until verification succeeds.

Program Pathways Mapper publishes the official 'how to finish your degree' roadmaps for hundreds of California community colleges, but only through a click-heavy, one-college-at-a-time web app with no API key and no export. This CLI mirrors a college's full catalog into local SQLite, then lets you - or an AI advisor - search across colleges, build a term-by-term plan with units rollups, compare two programs, find every program that requires a course, and diff catalog years. Offline, scriptable, and agent-native.

## When to Use This CLI

Use this CLI to query California Community College academic catalogs programmatically: resolve a college, mirror its programs/courses/maps offline, search across colleges, build term-by-term degree plans with unit totals, compare programs, find which programs require a course, diff catalog years, and pull CSU/UC transfer and career data. It is the right tool when an agent or script needs structured, joinable academic-pathway data that the ProgramMapper web app only exposes one click at a time.

## Anti-triggers

Do not use this CLI for:
- Do not use this CLI to register for classes or submit enrollment - it is read-only public catalog data, not a student information system.
- Do not use it to fetch official transcripts, grades, or any protected student records - ProgramMapper carries no student data.
- Do not use it for colleges outside California or outside the ProgramMapper network - it only covers CA community colleges in the registry.
- Do not treat course requisite text as a structured prerequisite graph - requisites are free-text descriptions, not machine edges.

## Unique Capabilities

These capabilities aren't available in any other tool for this API.

### Plan & compare from a local catalog
- **`plan`** — Build a program's full semester-by-semester plan with per-term and cumulative units, expanding required courses and 'choose one of' options.

  _When an agent is asked to build a 2-year academic plan, this returns the whole term structure with units in one structured call instead of paging a UI._

  ```bash
  programmapper-pp-cli plan 4a0cb2c2-b22f-324d-834e-80cbc2bde5f4 --agent
  ```
- **`compare`** — Show two programs side by side: shared courses, unique courses, and per-program units totals.

  _Lets a student or agent decide between two programs by the actual course overlap and unit cost, not prose._

  ```bash
  programmapper-pp-cli compare 4a0cb2c2-b22f-324d-834e-80cbc2bde5f4 7b7428c7-ab0e-4e00-b9eb-08091c31b34c --json
  ```
- **`diff-years`** — Diff a program's current map against a prior catalog year (2019-2025) and report added, removed, and changed courses plus unit deltas.

  _Answers 'did my program's requirements change since the catalog year I started under?' without manual side-by-side reading._

  ```bash
  programmapper-pp-cli diff-years 4a0cb2c2-b22f-324d-834e-80cbc2bde5f4 --from-year 2023 --json
  ```

### Reverse-lookups the website can't do
- **`course-programs`** — List every program and map at the synced college whose term plan includes a given course.

  _Answers a counselor's 'which programs require this course?' in one call instead of opening every map._

  ```bash
  programmapper-pp-cli course-programs "NURSING 090" --json
  ```
- **`bottlenecks`** — Rank courses across the synced college by how many program maps require them and by units, surfacing high-leverage and high-load courses.

  _Shows advisors which courses unlock the most programs so scheduling and tutoring effort goes where it matters._

  ```bash
  programmapper-pp-cli bottlenecks --limit 20 --json
  ```

### Cross-college & transfer
- **`search`** — Full-text search programs and courses across every synced college at once, offline.

  _Finds a program or course by keyword across many colleges in one query instead of visiting each college's site._

  ```bash
  programmapper-pp-cli search nursing --type programs --data-source local --limit 10
  ```
- **`transfer-options`** — Combine a program's CSU/UC linked transfer colleges and linked program maps into one view.

  _Gives an agent the full set of transfer destinations for a program in one structured response._

  ```bash
  programmapper-pp-cli transfer-options 4a0cb2c2-b22f-324d-834e-80cbc2bde5f4 --json
  ```

## HTTP Transport

This CLI uses Chrome-compatible HTTP transport for browser-facing endpoints. It does not require a resident browser process for normal API calls.

## Command Reference

**colleges** — California Community Colleges in the ProgramMapper registry

- `programmapper-pp-cli colleges get` — Get one college by its registry id (e.g. la_mission)
- `programmapper-pp-cli colleges list` — List every college in the registry (id, active site-content id, prior-year ids)
- `programmapper-pp-cli colleges resolve` — Resolve the college(s) for a ProgramMapper vanity URL

**courses** — Individual courses referenced by program maps

- `programmapper-pp-cli courses get` — Get a course's detail (units, description, requisites)
- `programmapper-pp-cli courses high-schools` — List high schools offering this course for dual enrollment

**interest-clusters** — Guided Pathways 'Areas of Interest' grouping programs (program-groups)

- `programmapper-pp-cli interest-clusters <site_content_id> <cluster_id>` — Get an interest cluster (program group) and the programs grouped under it

**meta** — API version and health

- `programmapper-pp-cli meta` — Get the ProgramMapper API version

**occupations** — Career and salary outcomes keyed by SOC code

- `programmapper-pp-cli occupations` — Batch-fetch career data (salary, job growth) for SOC codes

**program-maps** — Semester-by-semester course plans for a program pathway

- `programmapper-pp-cli program-maps get` — Get a program map: terms -> course/choice opportunities with units
- `programmapper-pp-cli program-maps linked` — Get a linked (transfer-partner) program map

**programs** — Degrees and certificates offered at a college

- `programmapper-pp-cli programs get` — Get a program's full detail (description, award, pathways, map refs)
- `programmapper-pp-cli programs linked` — Get a linked (transfer-partner) program
- `programmapper-pp-cli programs newer-year-link` — Get link data to the same program in a newer catalog year
- `programmapper-pp-cli programs pathways` — List a program's pathway summaries (each pathway is an alternate route)
- `programmapper-pp-cli programs prior-years` — List prior catalog years (2019+) that still have a published map for this program
- `programmapper-pp-cli programs search` — Full-text search programs by title and description

**site-contents** — Per-college, per-year catalog config (one site-content per college-year)

- `programmapper-pp-cli site-contents get` — Get a college-year's config (name, year, theme, titles)
- `programmapper-pp-cli site-contents home` — Get the home page: interest clusters (Areas of Interest), title, about

**transfer** — CSU/UC transfer pathways and 'choose one of' opportunities

- `programmapper-pp-cli transfer choice-opportunity` — Get a 'choose one of' opportunity (idealized program-pathway choice) detail
- `programmapper-pp-cli transfer linked-college` — Get a CSU/UC transfer college's detail for a transfer pathway


### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
programmapper-pp-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query.

## Recipes

### Mirror a college, then plan a degree

```bash
programmapper-pp-cli mirror la_mission --maps
```

Cache L.A. Mission programs, maps, and courses locally, then run plan/compare/course-programs offline.

### Narrow a deeply nested plan for an agent

```bash
programmapper-pp-cli plan 4a0cb2c2-b22f-324d-834e-80cbc2bde5f4 --agent --select terms.term_number,terms.items.code,terms.items.name,terms.items.min_units
```

Program maps are deeply nested; --select pulls only the term, course title, and units so an agent isn't flooded with the full map payload.

### Find which programs require a course

```bash
programmapper-pp-cli course-programs "NURSING 090" --json
```

Reverse-lookup every program at the mirrored college whose plan includes the course - impossible in the web UI.

### See what changed since your start year

```bash
programmapper-pp-cli diff-years 4a0cb2c2-b22f-324d-834e-80cbc2bde5f4 --from-year 2023
```

Diff the current Vocational Nursing map against the 2023 catalog to spot added, removed, or changed courses.

### Compare two programs head to head

```bash
programmapper-pp-cli compare 4a0cb2c2-b22f-324d-834e-80cbc2bde5f4 7b7428c7-ab0e-4e00-b9eb-08091c31b34c --json
```

Show shared vs. unique courses and total units for two programs side by side.

## Auth Setup

No authentication required.

Run `programmapper-pp-cli doctor` to verify setup.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color --yes`.

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  programmapper-pp-cli colleges list --agent --select id,name,status
  ```
- **Previewable** — `--dry-run` shows the request without sending
- **Offline-friendly** — sync/search commands can use the local SQLite store when available
- **Non-interactive** — never prompts, every input is a flag
- **Explicit retries** — use `--idempotent` only when an already-existing create should count as success

### Response envelope

Commands that read from the local store or the API wrap output in a provenance envelope:

```json
{
  "meta": {"source": "live" | "local", "synced_at": "...", "reason": "..."},
  "results": <data>
}
```

Parse `.results` for data and `.meta.source` to know whether it's live or local. A human-readable `N results (live)` summary is printed to stderr only when stdout is a terminal AND no machine-format flag (`--json`, `--csv`, `--compact`, `--quiet`, `--plain`, `--select`) is set — piped/agent consumers and explicit-format runs get pure JSON on stdout.

## Paths and state

Agents should treat the CLI's path resolver as part of the runtime contract:

- Use `--home <dir>` for one invocation, or set `PROGRAMMAPPER_HOME=<dir>` to relocate all four path kinds under one root.
- Use per-kind env vars only when a specific kind must diverge: `PROGRAMMAPPER_CONFIG_DIR`, `PROGRAMMAPPER_DATA_DIR`, `PROGRAMMAPPER_STATE_DIR`, `PROGRAMMAPPER_CACHE_DIR`.
- Resolution order is per-kind env var, `--home`, `PROGRAMMAPPER_HOME`, XDG (`XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, `XDG_CACHE_HOME`), then platform defaults.
- `config` contains settings like `config.toml` and profiles. `data` contains `credentials.toml`, `data.db`, cookies, and auth sidecars. `state` contains persisted queries, jobs, and `teach.log`. `cache` contains regenerable HTTP/cache files.
- Stored secrets live in `credentials.toml` under the data dir. Existing legacy `config.toml` secrets are read for compatibility and leave `config.toml` on the first auth write.
- Run `programmapper-pp-cli doctor --fail-on warn` to surface path and credential-location warnings. `agent-context` exposes a schema v4 `paths` block for agents that need the resolved dirs.
- For MCP, pass relocation through the MCP host config. The MCP binary does not inherit CLI flags:

  ```json
  {
    "mcpServers": {
      "programmapper": {
        "command": "programmapper-pp-mcp",
        "env": {
          "PROGRAMMAPPER_HOME": "/srv/programmapper"
        }
      }
    }
  }
  ```

Fleet precedence: an inherited per-kind env var overrides an explicit `--home` for that kind. Use `PROGRAMMAPPER_HOME` or per-kind vars as durable fleet levers, and use `--home` only for a single invocation. Relocation is not reversible by unsetting env vars; move files manually before clearing `PROGRAMMAPPER_HOME`, or `doctor` will not find credentials left under the former root.

## Agent Feedback

When you (or the agent) notice something off about this CLI, record it:

```
programmapper-pp-cli feedback "the --since flag is inclusive but docs say exclusive"
programmapper-pp-cli feedback --stdin < notes.txt
programmapper-pp-cli feedback list --json --limit 10
```

Entries are stored locally as `feedback.jsonl` under the resolved data dir. They are never POSTed unless `PROGRAMMAPPER_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `PROGRAMMAPPER_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

Write what *surprised* you, not a bug report. Short, specific, one line: that is the part that compounds.

## Output Delivery

Every command accepts `--deliver <sink>`. The output goes to the named sink in addition to (or instead of) stdout, so agents can route command results without hand-piping. Three sinks are supported:

| Sink | Effect |
|------|--------|
| `stdout` | Default; write to stdout only |
| `file:<path>` | Atomically write output to `<path>` (tmp + rename) |
| `webhook:<url>` | POST the output body to the URL (`application/json` or `application/x-ndjson` when `--compact`) |

Unknown schemes are refused with a structured error naming the supported set. Webhook failures return non-zero and log the URL + HTTP status on stderr.

## Named Profiles

A profile is a saved set of flag values, reused across invocations. Use it when a scheduled agent calls the same command every run with the same configuration - HeyGen's "Beacon" pattern.

```
programmapper-pp-cli profile save briefing --json
programmapper-pp-cli --profile briefing colleges list
programmapper-pp-cli profile list --json
programmapper-pp-cli profile show briefing
programmapper-pp-cli profile delete briefing --yes
```

Explicit flags always win over profile values; profile values win over defaults. `agent-context` lists all available profiles under `available_profiles` so introspecting agents discover them at runtime.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage error (wrong arguments) |
| 3 | Resource not found |
| 5 | API error (upstream issue) |
| 7 | Rate limited (wait and retry) |
| 10 | Config error |

## Argument Parsing

Parse `$ARGUMENTS`:

1. **Empty, `help`, or `--help`** → show `programmapper-pp-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → see Prerequisites above
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## MCP Server Installation

1. Install the MCP server:
   ```bash
   go install github.com/mvanhorn/printing-press-library/library/other/programmapper/cmd/programmapper-pp-mcp@latest
   ```
2. Register with Claude Code:
   ```bash
   claude mcp add programmapper-pp-mcp -- programmapper-pp-mcp
   ```
3. Verify: `claude mcp list`

## Direct Use

1. Check if installed: `which programmapper-pp-cli`
   If not found, offer to install (see Prerequisites at the top of this skill).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   programmapper-pp-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `programmapper-pp-cli <command> --help`.
