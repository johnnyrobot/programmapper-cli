# ProgramMapper Novel-Features Brainstorm (subagent audit trail)

## Customer model

**Maya — first-generation transfer-bound student (L.A. Mission College, full-time).** Wants an ADT in Biology to CSUN; reads term-by-term maps one semester at a time in a tab she loses; can't compare two program maps side by side, can't total remaining units, can't tell what changed vs. the year she started.

**Diego — academic counselor / articulation officer (~350 advisees).** Answers "which programs require CHEM 101?" by opening each map and scanning visually because the SPA only navigates program→map, never map→course→program. Preps 8–10 plans/week, re-hunting bottleneck courses and transfer destinations. No reverse lookup, no catalog-year diff.

**Aria — AI advising agent (LLM building a 2-year plan in a chatbot).** Asked to "build a 2-year plan for a Nursing transfer to CSUN" but the only source is a click-heavy SPA with no API key/export, so it hallucinates or refuses. Needs structured, joinable JSON: programs, term plan, per-course units, transfer targets, salary outlook — in one shot.

## Survivors (>=5/10)

| # | Feature | Command | Score | Buildability |
|---|---------|---------|-------|--------------|
| 1 | Cross-program compare | `compare <programId> <programId>` | 8/10 | hand-code |
| 2 | Course→program reverse lookup | `course programs <courseId>` | 8/10 | hand-code |
| 3 | Term/units planner | `plan <programId>` | 9/10 | hand-code |
| 4 | Catalog-year diff | `diff-years <programId>` | 8/10 | hand-code |
| 5 | Multi-college search | `search --type program/course <q>` | 8/10 | (framework search) |
| 6 | Transfer rollup | `transfer <programId>` | 7/10 | hand-code |
| 7 | Bottleneck courses | `bottlenecks` | 7/10 | hand-code |

## Killed candidates
- `careers` — value contingent on programs carrying SOC codes; reachable via generated `occupations careers`.
- `choices` — `plan` already expands CHOICE opportunities.
- `units` — subset of `plan`'s units rollup.
- `catalog` — thin over generated `site-contents home` + cluster browse.
- `prereqs` — requisites are free-text, not structured edges; needs NLP.
- `bundle` — meta-wrapper; `plan --agent` serves the agent persona.
- `stale` — narrow ops-hygiene; lowest user-pain of the local-query set.
