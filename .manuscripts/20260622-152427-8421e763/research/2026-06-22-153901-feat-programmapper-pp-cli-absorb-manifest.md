# ProgramMapper CLI — Absorb Manifest

**API:** Program Pathways Mapper (`https://b.api.programmapper.com`) — no auth, Surf transport.
**Competing tools found:** none (no CLI, SDK, npm package, or MCP server exists). The
"absorb" surface is parity with the ProgramMapper web UI itself; differentiation is in
transcendence (local SQLite mirror + cross-entity queries the SPA cannot do).

## Absorbed (match the web UI — all generated-endpoint commands)
| # | Feature | Best Source | Our Implementation |
|---|---------|-------------|--------------------|
| 1 | Resolve a college by vanity URL | ProgramMapper web UI | (generated endpoint) colleges resolve |
| 2 | List all 954 colleges | ProgramMapper web UI | (generated endpoint) colleges list |
| 3 | Get one college by id | ProgramMapper web UI | (generated endpoint) colleges get |
| 4 | View college-year config (name, year, theme) | ProgramMapper web UI | (generated endpoint) site-contents get |
| 5 | Browse interest clusters (home page) | ProgramMapper web UI | (generated endpoint) site-contents home |
| 6 | View an interest cluster's programs | ProgramMapper web UI | (generated endpoint) interest-clusters get |
| 7 | Full-text program search (server-side, one college) | ProgramMapper web UI | (generated endpoint) programs search |
| 8 | View a program (desc, award, pathways) | ProgramMapper web UI | (generated endpoint) programs get |
| 9 | View a program's pathways | ProgramMapper web UI | (generated endpoint) programs pathways |
| 10 | Prior-year maps for a program | ProgramMapper web UI | (generated endpoint) programs prior-years |
| 11 | Newer-year link data for a program | ProgramMapper web UI | (generated endpoint) programs newer-year-link |
| 12 | View a program map (term-by-term) | ProgramMapper web UI | (generated endpoint) program-maps get |
| 13 | View a linked (transfer) program map | ProgramMapper web UI | (generated endpoint) program-maps linked |
| 14 | View a linked (transfer) program | ProgramMapper web UI | (generated endpoint) programs linked |
| 15 | View a course (units, description) | ProgramMapper web UI | (generated endpoint) courses get |
| 16 | Dual-enrollment high schools for a course | ProgramMapper web UI | (generated endpoint) courses high-schools |
| 17 | CSU/UC transfer college detail | ProgramMapper web UI | (generated endpoint) transfer linked-college |
| 18 | "Choose one of" choice opportunity detail | ProgramMapper web UI | (generated endpoint) transfer choice-opportunity |
| 19 | Career/salary data by SOC code | ProgramMapper web UI | (generated endpoint) occupations careers |
| 20 | API version / health | ProgramMapper web UI | (generated endpoint) meta info |
| 21 | Offline catalog mirror (sync) | (framework) | (behavior in programmapper-pp-cli sync) local SQLite mirror of colleges/programs/courses/maps |
| 22 | Cross-college FTS (programs + courses) | (framework) | (behavior in programmapper-pp-cli search) FTS5 across all synced colleges; `--type program`/`--type course` |

## Transcendence (only possible with our local-mirror approach)
| # | Feature | Command | Buildability | Why Only We Can Do This | Long Description |
|---|---------|---------|--------------|------------------------|------------------|
| 1 | Term/units planner | plan | hand-code | Walks synced map terms→opportunities, expands COURSE+CHOICE, rolls up per-term & cumulative min/max units — no single API call returns this | Use this for the full term-by-term plan WITH units rollups and choice expansion. Do NOT use it to fetch the raw map JSON for one map id; use 'program-maps get'. |
| 2 | Cross-program compare | compare | hand-code | SQLite join across program/map/opportunity/course to diff two programs' course sets and units — the SPA can't show two maps side by side | none |
| 3 | Course→program reverse lookup | course-programs | hand-code | Reverse index over local opportunity→course rows; the SPA only navigates program→map→course, never the reverse | Use this to find every program that REQUIRES a course (reverse lookup). Do NOT use it to fetch one course's units/description; use 'courses get'. |
| 4 | Catalog-year diff | diff-years | hand-code | Joins current vs. prior-year (2019–2025) map snapshots in SQLite and reports added/removed/changed courses and unit deltas | none |
| 5 | Transfer rollup | transfer-options | hand-code | Joins local linked-programs + linked-transfer-colleges into one CSU/UC transfer view the SPA shows one-at-a-time | Use this for a program's combined CSU/UC transfer destinations and linked maps. Do NOT use it to fetch one transfer-college record by id; use 'transfer linked-college'. |
| 6 | Bottleneck courses | bottlenecks | hand-code | Aggregates the local opportunity→course index ranking courses by how many maps require them and by units; no API aggregation endpoint exists | none |

**Hand-code count:** 6 transcendence commands (each ~50–150 LoC + root.go wiring).
**Framework-delivered transcendence:** offline `sync` + cross-college `search` (multi-college FTS) come from the generator.
**No stubs.** All 6 transcendence rows are shipping scope.
