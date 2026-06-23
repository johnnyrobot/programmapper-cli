# Phase 4.95 Local Code Review — findings

Reviewer: general-purpose Go reviewer over hand-written files (pmstore.go, mirror.go, plan.go, compare.go, diff_years.go, course_programs.go, bottlenecks.go, transfer_options.go). Scope excluded generated files and internal/cliutil, internal/mcp/cobratree.

High: none.

Autofixed in-place (med + select low), then rebuilt + re-verified:
- pmstore.go: pmGetProgram / pmGetMap now branch on errors.Is(err, sql.ErrNoRows) and propagate real store errors instead of swallowing them as "not found".
- pmstore.go: pmIsRateLimited now uses typed `*client.APIError` StatusCode (429/403), with the string match as a fallback.
- mirror.go: site-content fetch non-rate-limit error now warns instead of silently dropping; map upsert error now warns + skips the count increment (no overstated maps count).
- bottlenecks.go: course units now tracked as min/max across every map that uses the course (was last-writer-wins); zero-unit courses no longer dropped.
- diff_years.go: prior years sorted newest-first before default selection (no reliance on API order); text render labels the value "min-unit delta".
- transfer_options.go: career-data unmarshal failure no longer silently flips source to "live" and drops data — sets an explanatory note and keeps source local.
- course_programs.go: removed an empty `if hintIfUnsynced {}` branch.

Reviewed and intentionally not changed:
- `--agent` JSON-gating "inconsistency": false positive — `--agent` sets `flags.asJSON=true` in root PreRun, which the `flags.asJSON` predicate already covers.
- diff_years.go `firstNonEmpty(SiteContentID, ID)` fallback: the live prior-years response shows entry `id` == `pathwaySummary.siteContentId`, so the fallback resolves to the correct prior site-content id.
- mentionsTransfer / trimFloat heuristics: bounded inputs, low risk.

Clean per reviewer: no nil-deref / index-out-of-range (range-based access, typed Unmarshal), store/context resources deferred-closed on all paths, mirror is single-goroutine (no races), no SQL/path injection (typed store API; server-issued ids in URL paths only).

Convergence: findings cleared in 1 round; build + vet + go test green; touched commands re-verified live.

/simplify: skipped — working tree is under the run-state dir (not a git repo), so the diff-scoped simplifier has no changeset to operate on; autofix output was minimal and already reviewer-clean.
