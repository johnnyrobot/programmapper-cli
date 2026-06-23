# ProgramMapper Discovery Report

**Target:** `https://la-mission.programmapper.ws/academics` (Angular SPA)
**Backend API:** `https://b.api.programmapper.com`
**Method:** static JS-bundle analysis + live probing (curl + Surf + headless Chrome page-context fetch)

## Reachability
- `probe-reachability https://b.api.programmapper.com/info` → **`browser_http`** (conf 0.85):
  - stdlib HTTP → **403** (`403/429 HTML or access-denied`)
  - surf-chrome (Chrome TLS fingerprint) → **200 application/json**
  - `needs_browser_capture: false`, `needs_clearance_cookie: false`
- **Runtime decision:** printed CLI ships **Surf transport** (`http_transport: browser-chrome`). No clearance cookie, no resident browser.
- **WAF behavior:** per-IP burst rate limiting. A ~15-request curl burst plus headless-Chrome eval bursts pushed this host IP into a sustained 403 window covering *both* hosts. Normal paced Surf traffic from the CLI clears it. The CLI must surface a typed rate-limit error (429/403), never empty results.

## Config (from `/assets/env-specific.json`)
```json
{"apiBaseUrl": "https://b.api.programmapper.com", "previewMode": false, "gtmAccount": "GTM-54PHRX6D"}
```

## Auth
None. No `Authorization`/cookie/token in `apiUrlService.getRestData` (plain `http.get`). Public student-facing data. CORS allows the per-college SPA origin.

## Endpoint inventory (definitive — from `main.bundle.js` `getRestData`/`postRestData` call sites)
GET unless noted. `{scid}` = site-content id (college-year).

| Method | Path | Notes |
|---|---|---|
| GET | `info` | `{version}` — health/version |
| GET | `colleges` | 954 college records `{id, activeSiteContentId, latestSiteContentIdsByYear, csu, uc, accessibleTextColor}` |
| GET | `colleges?url={vanityUrl}` | resolve college by hostname (needs scheme: `https://la-mission.programmapper.ws`) |
| GET | `colleges/{id}` | one college |
| GET | `site-contents/{scid}` | college-year config `{collegeName, collegeId, year, programMapTitle, homePageTitle, applyNowUrl, collegeLogo, themeColor, tag, showPriorMapYears, excludedPriorMapYears, id}` |
| GET | `site-contents/{scid}/home-page-content` | `{title, about, programGroups[], siteContentId}` — interest clusters |
| GET | `site-contents/{scid}/interest-clusters/{groupId}` | cluster detail → programs |
| POST | `site-contents/{scid}/programs/search` | body `{searchString}` → `{programs:[{programId,title,award,truncatedDescription}], resultCount}` |
| GET | `site-contents/{scid}/programs/{programId}` | full program |
| GET | `site-contents/{scid}/programs/{programId}/pathways` | pathway summaries |
| GET | `site-contents/{scid}/programs/{programId}/prior-years-with-maps` | `?requestedProgramMapId=` |
| GET | `site-contents/{scid}/programs/{programId}/newer-year-link-data` | `?requestedProgramMapId=&requestedLinkedProgramMapId=` |
| GET | `site-contents/{scid}/program-maps/{mapId}` | term-by-term map |
| GET | `site-contents/{scid}/linked-program-maps/{mapId}` | transfer-partner map |
| GET | `site-contents/{scid}/linked-programs/{programId}` | linked program |
| GET | `site-contents/{scid}/courses/{courseId}` | course detail |
| GET | `site-contents/{scid}/courses/{courseId}/highSchools` | dual-enrollment high schools |
| GET | `site-contents/{scid}/choice-opportunities/{id}` | "choose one of" detail |
| GET | `site-contents/{scid}/linked-transfer-colleges/{id}` | CSU/UC transfer college |
| POST | `standard-occupations/batch` | body `{standardOccupationalCodes:[...]}` → career data (salary, job-growth) |

## Verified live response samples (captured before IP block)
- `info` → `{"version":"9.0.81"}`
- `colleges?url=https://la-mission.programmapper.ws` → `[{"id":"la_mission","activeSiteContentId":"1969dfb0-9adc-4768-9385-2eda17d89839","latestSiteContentIdsByYear":{"2019":"...","2020":"...","2021":"...","2022":"...","2023":"...","2024":"...","2025":"1969dfb0-..."},"csu":false,"uc":false,"accessibleTextColor":false}]`
- `site-contents/1969dfb0-...` → `{"collegeName":"L.A. Mission College","collegeId":"la_mission","year":2025,"programMapTitle":"Program Map","homePageTitle":"Program Mapper","applyNowUrl":null,"themeColor":"#334c74","tag":"13",...}`
- `home-page-content` → `{"programGroups":[6 × {title,icon,description,masterRecordId}],"title":...,"about":...,"siteContentId":...}` — clusters: Arts/Media/Performance, Business/Law/Public Safety, Child/Family/Education, Culinary Arts, Society/Culture/Communication, STEM/Health/Fitness
- `programs/search {"searchString":"nursing"}` → `{"programs":[{"programId":"4a0cb2c2-...","title":"Vocational Nursing Training Program","award":"Certificate of Achievement","truncatedDescription":"..."}],"resultCount":1}`

## Field hints from bundle (for endpoints not sampled live)
- program-pathway-card: `minUnits, maxUnits, minHours, maxHours, opportunity.type ("COURSE"), term.termNumber, requirement, mappedCourseId, pathwayElements[].hasPriorLearningCredit`
- careers (`standard-occupations/batch`): `standardOccupationalCodes, careerJobGrowthPct, careerAverageSalary, careerLowSalary, careerHighSalary, highSalaryMaxed`
- course / high-school address: `lineOne, lineTwo, city, state, zip`
- pathway: `pathwayId, linkedPathwayId, pathwayElements[]`

## Replayability
PASS — pure JSON REST over HTTPS, replayable via Surf (Chrome TLS fingerprint). No resident browser, no page-context execution required.
