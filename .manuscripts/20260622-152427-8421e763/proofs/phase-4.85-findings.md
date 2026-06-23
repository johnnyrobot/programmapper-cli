# Phase 4.85 Agentic Output Review — findings (Wave B, warnings-only)

Manual + scorecard live-check sampling of every novel command's real output:

| Command | Sampled output | Plausibility verdict |
|---------|----------------|----------------------|
| plan <nursing> | 4 terms, NURSING 090/091/092 (MAJOR_CORE), 37.5 units | RELEVANT — courses match the program |
| compare nursing vs biology | 8 vs 7 courses, 0 shared, correct only-A/only-B sets | RELEVANT — distinct vocational vs transfer programs |
| diff-years biology 2022 | +CHEM 065, -ENGLISH 101/MATH 261, -4 units | RELEVANT — real catalog-year deltas |
| course-programs "ART 201" | 2 programs (Art History term 1), scanned 8 maps | RELEVANT — reverse lookup correct |
| bottlenecks | ART 201/ARTHIST 120 required by 2 programs each | RELEVANT — frequency ranking correct |
| transfer-options biology | transfer-designated, "to CSU - Cal-GETC", avg $112k, 6.1% growth, 9 careers | RELEVANT — transfer+career join correct |
| search nursing --type programs --data-source local | 2 nursing programs by title | RELEVANT — FTS relevance correct, no substring-match noise |

No substring-match relevance failures, no format bugs, no silent source drops observed.

**Note on scorecard --live-check (1/7 pass):** the probe runs each novel command WITHOUT first running `mirror`. These commands are mirror-dependent by design (a bare programId must be resolved to its site-content via the mirror or `--site-content`). With no mirror they return an honest empty result + a "run mirror" hint, which the token-match probe scores as a miss. This is the probe environment, not a defect — the table above confirms correct output once a mirror exists.

Verdict: PASS (no blocking findings).
