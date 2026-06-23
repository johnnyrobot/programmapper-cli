# ProgramMapper CLI — Phase 3 Build Log

Manifest transcendence rows: 6 planned, 0 built. Phase 3 will not pass until all 6 ship.

Planned hand-code transcendence commands (each ~50-150 LoC + root.go wiring, all pre-wired by the generator as stubs):
- plan (newNovelPlanCmd)
- compare (newNovelCompareCmd)
- diff-years (newNovelDiffYearsCmd)
- course-programs (newNovelCourseProgramsCmd)
- bottlenecks (newNovelBottlenecksCmd)
- transfer-options (newNovelTransferOptionsCmd)

Plus foundation (Priority 0): hand-built `mirror` command (offline catalog graph walk) + shared `pmstore.go` helpers. The framework `search` and `sync` cover cross-college FTS; the nested ProgramMapper graph (no flat list endpoints) requires the custom mirror.

Spec fix applied before Phase 3: interest-clusters endpoint path corrected from /interest-clusters/{id} (a front-end route, 404) to /program-groups/{id} (the real API path). Regenerated clean.

## Phase 3 complete

Manifest transcendence rows: 6 planned, 6 built. ALL ship.
- plan ✓  compare ✓  diff-years ✓  course-programs ✓  bottlenecks ✓  transfer-options ✓
Framework-delivered: cross-college search (local FTS), offline mirror.

Foundation (Priority 0): `mirror` command (paced, capped, resumable, rate-limit-aware graph walk) + `pmstore.go` shared helpers (resolve, load, models). SaveSyncState recorded so store reports synced.

Behavioral acceptance (live, real L.A. Mission data):
- mirror la_mission: 6 clusters, 161 programs (light); --maps --max-programs 8: 8 maps, 29 courses
- plan <nursing>: 4 terms, 37.5 units, NURSING 090 MAJOR_CORE in term 1 (map fetched live)
- compare <nursing> <biology>: 8 vs 7 courses, 0 shared, 8 only-A, 7 only-B
- diff-years <biology> --from-year 2022: +CHEM 065, -ENGLISH 101/MATH 261, -4 units
- course-programs "ART 201": 2 programs (Art History, term 1) across 8 scanned maps
- bottlenecks: ART 201/ARTHIST 120 each required by 2 programs
- transfer-options <biology>: transfer-designated, "to CSU - Cal-GETC", career avg $112k, 6.1% growth, 9 careers
- search nursing --type programs --data-source local: 2 programs, no unsynced hint

Gate: per-row Cobra resolution 7/7 PASS; dogfood novel_features_check 7/7 found, 0 missing, not skipped; tests: transcend_test.go (real table-driven) + 6 behavior tests (t.Skip replaced).

## Known generator mismatch (retro candidate)
Framework `search.go` hard-codes a per-tenant search endpoint (POST /site-contents/{site_content_id}/programs/search with a literal placeholder) as its "live search". ProgramMapper's search is per-site-content and needs a scid+body, so the framework's auto-mode live search is broken. Worked around by documenting `--data-source local` (the cross-college search is inherently offline FTS). Generated search.go left unedited to stay regen-safe.

Spec param-name note: `interest-clusters` endpoint path was a front-end route (/interest-clusters/{id}, 404); corrected to the API path (/program-groups/{id}) and regenerated before Phase 3.
