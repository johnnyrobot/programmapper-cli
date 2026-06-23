# Program Pathways Mapper (ProgramMapper) CLI Brief

## API Identity
- **Product:** Program Pathways Mapper (PPM) — public, student-facing academic
  pathway maps for California Community Colleges. Powered by Bakersfield College /
  Kern CCD, the CA Community Colleges Chancellor's Office, and FoundationCCC.
- **Front-end:** Angular SPA served per-college at `<college>.programmapper.ws`
  (target: `la-mission.programmapper.ws` → L.A. Mission College, 2025 catalog).
- **Backend API:** `https://b.api.programmapper.com` — REST/JSON, **no auth**,
  multi-tenant (one `site-content` per college-year). 954 college records in the
  registry (includes dev/qa variants).
- **Data profile:** colleges → site-contents (college-year) → interest clusters
  (Guided Pathways "Areas of Interest") → programs (degrees/certificates) →
  pathways → program-maps (term-by-term course plans) → courses; plus
  transfer links (CSU/UC, assist.org) and career data (salary/job-growth by SOC code).

## Reachability Risk
- **Mode:** `browser_http` (probe-reachability, confidence 0.85). Plain Go stdlib
  HTTP → **403**; **Surf with a Chrome TLS fingerprint → 200 application/json**.
  The WAF blocks on TLS fingerprint, not on auth. `needs_clearance_cookie: false`,
  `needs_browser_capture: false`.
- **Runtime decision (settled):** the printed CLI MUST ship **Surf transport**
  (`http_transport: browser-chrome` in the spec). No clearance cookie, no resident
  browser. Fully replayable.
- **Per-IP rate limiting:** the WAF aggressively throttles bursts from a single IP
  (a ~15-request curl burst tripped a sustained 403 window). The printed CLI must
  pace requests and surface a typed rate-limit error on 429/403, not empty results.
- Probe-safe endpoint used: `GET /info` → `{"version":"9.0.81"}` (200 via Surf).

## Top Workflows
1. **Browse programs by interest cluster** — see the 6 Areas of Interest and the
   degrees/certificates under each (Arts/Media, Business/Law, Child/Family/Education,
   Culinary Arts, Society/Culture/Communication, STEM/Health/Fitness).
2. **Search programs** — full-text search returning `{programId, title, award,
   truncatedDescription}` for matching degrees/certificates.
3. **View a program's map** — the semester-by-semester course plan (terms → courses,
   units, requirement groups) that is the product's headline artifact.
4. **Inspect a course** — units, description, requisites, which programs/maps use it.
5. **Transfer & career planning** — CSU/UC transfer links per program; career
   outcomes (salary, job-growth) by SOC code via `standard-occupations/batch`.
6. **Compare catalog years** — each program has prior-year maps (2019–2025), so a
   student can see how requirements changed.

## Table Stakes
- No competing CLI, MCP server, SDK, or wrapper exists for ProgramMapper (verified:
  the product is niche CA-CCC infrastructure; `programmapper.org` is the marketing
  site). This CLI is **novel** — the "absorb" surface is empty; value is in
  transcendence (offline mirror, search, compare, plan).
- Implicit table stakes from the web UI: program browse, program search, program map
  view, course detail, transfer info, career data, prior-year maps, multi-college
  (any `<college>.programmapper.ws` via the colleges registry).

## Data Layer
- **Primary entities:** college, site-content (college-year config), interest-cluster
  (program group), program, pathway, program-map (terms + opportunities), course,
  linked-transfer-college, career/occupation.
- **Local store value:** the full catalog for a college fits easily in SQLite. A
  local mirror enables offline search, cross-program comparison, course→program
  reverse lookup, units rollups, and term-by-term planning — none of which the web
  UI or a single API call provides.
- **Sync key:** site-content id (per college-year). `colleges?url=<vanity>` resolves
  the active + prior-year site-content ids.
- **FTS/search:** programs (title, award, description) and courses (code, title,
  description) → FTS5.

## Codebase Intelligence (static, from the SPA bundle — definitive)
- API base resolved at runtime from `/assets/env-specific.json`:
  `{"apiBaseUrl":"https://b.api.programmapper.com"}`.
- All data goes through `apiUrlService.getRestData(path)` → `GET apiBaseUrl + "/" + path`.
  Two POSTs: `programs/search` and `standard-occupations/batch`.
- **Endpoint map (definitive, from `main.bundle.js`):**
  | Method | Path | Returns |
  |---|---|---|
  | GET | `info` | `{version}` |
  | GET | `colleges` | array of `{id, activeSiteContentId, latestSiteContentIdsByYear, csu, uc, accessibleTextColor}` (954) |
  | GET | `colleges?url=<vanityUrl>` | the college record(s) for a hostname (needs scheme, e.g. `https://la-mission.programmapper.ws`) |
  | GET | `colleges/{id}` | one college by id |
  | GET | `site-contents/{scid}` | college-year config: `{collegeName, collegeId, year, programMapTitle, homePageTitle, applyNowUrl, collegeLogo, themeColor, tag, showPriorMapYears, excludedPriorMapYears, id}` |
  | GET | `site-contents/{scid}/home-page-content` | `{title, about, programGroups[], siteContentId}` (interest clusters) |
  | GET | `site-contents/{scid}/interest-clusters/{groupId}` | cluster detail → programs in the cluster |
  | POST | `site-contents/{scid}/programs/search` `{searchString}` | `{programs:[{programId,title,award,truncatedDescription}], resultCount}` |
  | GET | `site-contents/{scid}/programs/{programId}` | full program (description, award, pathways, map refs) |
  | GET | `site-contents/{scid}/programs/{programId}/pathways` | pathway summaries |
  | GET | `site-contents/{scid}/programs/{programId}/prior-years-with-maps` | prior catalog years that have maps (`?requestedProgramMapId=`) |
  | GET | `site-contents/{scid}/programs/{programId}/newer-year-link-data` | newer-year link data |
  | GET | `site-contents/{scid}/program-maps/{mapId}` | program map: terms → opportunities (courses/choices), units |
  | GET | `site-contents/{scid}/linked-program-maps/{mapId}` | linked (transfer-partner) program map |
  | GET | `site-contents/{scid}/linked-programs/{programId}` | linked program |
  | GET | `site-contents/{scid}/courses/{courseId}` | course detail (units, description, requisites) |
  | GET | `site-contents/{scid}/courses/{courseId}/highSchools` | high schools offering the course (dual-enrollment) |
  | GET | `site-contents/{scid}/choice-opportunities/{id}` | "choose one of" opportunity detail |
  | GET | `site-contents/{scid}/linked-transfer-colleges/{id}` | CSU/UC transfer college detail |
  | POST | `standard-occupations/batch` `{standardOccupationalCodes:[...]}` | career data (salary, job-growth) by SOC code |
- Auth: none observed. CORS allows the per-college SPA origin.
- Front-end router routes (`/academics/programs/`, `/academics/interest-clusters/{id}`,
  `/academics/search-results`) are NOT API paths — don't model them.

## User Vision
- (none provided — user chose "Let's go")

## Product Thesis
- **Name:** `programmapper` (CLI `programmapper-cli`)
- **Why it should exist:** ProgramMapper is the official map of *how to actually
  finish a degree* at a CA community college — which courses, in which order, with
  transfer and career context. Today that lives behind a click-heavy Angular SPA,
  one college at a time, with no API key, no export, no search across programs, and
  no way for an academic-advising agent to query it. A CLI with a local SQLite mirror
  turns the entire catalog into something an AI counselor or a student can query,
  compare, and plan against — offline, scriptable, agent-native. It's the difference
  between "browse the website" and "build me a 2-year plan for a Nursing transfer to
  CSUN."

## Build Priorities
1. **Foundation:** Surf transport (`http_transport: browser-chrome`); local SQLite
   mirror of colleges, site-contents, programs, courses, maps; `sync`; FTS search.
2. **Absorb (parity with the web UI):** college resolve, interest-cluster browse,
   program search, program detail, program-map view, course detail, transfer links,
   career data, prior-year maps.
3. **Transcend (offline-only / agent-native):** cross-program compare, course→program
   reverse lookup, term/units planner, "what changed" year diff, multi-college search,
   transfer/career rollups. (Full list in the absorb manifest.)
