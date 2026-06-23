# ProgramMapper CLI — Shipcheck

## Verdict: ship

shipcheck umbrella: **PASS (7/7 legs)**.

| Leg | Result | Notes |
|-----|--------|-------|
| verify | PASS | mock-server runtime, no auth |
| validate-narrative | PASS | 10 narrative commands resolved + full examples passed |
| dogfood | PASS | novel_features_check 7/7 found, 0 missing; wiring OK |
| workflow-verify | PASS | workflow-pass |
| apify-audit | PASS | n/a |
| verify-skill | PASS | flag-names, flag-commands, positional-args, canonical-sections all pass |
| scorecard | PASS | **92/100 Grade A** |

## Scorecard highlights
- Output Modes 10, Auth 10, Error Handling 10, Terminal UX 10, README 10, Doctor 10, Agent Native 10, MCP Quality 10, MCP Remote Transport 10, Local Cache 10, Workflows 10, Breadth 9, Vision 9, Agent Workflow 9, Path Validity 10, Sync Correctness 10, Dead Code 5/5.
- Sub-65 dims (non-blocking; total 92): Insight 4/10, Cache Freshness 5/10, MCP Desc Quality 7/10, MCP Token Efficiency 7/10, Type Fidelity 4/5.

## Behavioral correctness (verified live, real L.A. Mission data)
All 7 novel features produce correct output WITH a mirror (see build log). The scorecard `--live-check` sample probe passed 1/7 because it runs each novel command **without first running `mirror`** — these commands are mirror-dependent by design (ProgramMapper has no flat program/course lookup; a programId must be resolved to its site-content via the mirror or `--site-content`). With no mirror, each command returns an honest empty result plus an actionable "run mirror" hint (verified), which the token-match probe scores as a miss. This is a probe-environment limitation, not a feature defect; Phase 5 live dogfood mirrors first and exercises real behavior.

## Fixes applied during shipcheck
- Recipe `mirror ... && plan ...` compound -> single `mirror la_mission --maps` (validate-narrative/Phase 4.9 friendliness).
- `course-programs <courseId>` placeholder -> real `course-programs "NURSING 090"` across research.json, SKILL.md, README.md.
- `--select` recipe paths corrected to the real plan JSON shape (`terms.term_number,terms.items.code,terms.items.name,terms.items.min_units`).
- mirror records `sync_state` so the "not synced" hint stops after a mirror.

## Known gap (documented, non-blocking)
Framework `search` auto-mode targets a per-tenant search endpoint it cannot parameterize; the cross-college search is offline FTS, invoked with `--data-source local` (documented in all examples). Generated `search.go` left unedited to stay regen-safe. Logged as a retro candidate.

Ship recommendation: **ship**.
