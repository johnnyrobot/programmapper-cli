// Copyright 2026 johnnyrobot and contributors. Licensed under Apache-2.0. See LICENSE.
//
// pmstore.go — shared helpers and typed models for ProgramMapper's transcendence
// commands (mirror, plan, compare, diff-years, course-programs, bottlenecks,
// transfer-options). Hand-authored; not generated.
//
// ProgramMapper has no flat list endpoints: every program/course/map lives under
// site-contents/{scid}/... and must be reached by id. The `mirror` command walks
// that graph into the local SQLite store; the transcendence commands read it.

package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/johnnyrobot/programmapper-cli/internal/client"
	"github.com/johnnyrobot/programmapper-cli/internal/store"
)

// Local-store resource_type keys. These land in the generic `resources` table
// (and resources_fts), so the framework `search` command finds programs and
// courses with `--type programs` / `--type courses`.
const (
	rtSiteContents = "site_contents"
	rtClusters     = "interest_clusters"
	rtPrograms     = "programs"
	rtProgramMaps  = "program_maps"
	rtCourses      = "courses"
)

// ---- Typed models (fields confirmed against live L.A. Mission responses) ----

type pmCollege struct {
	ID                         string            `json:"id"`
	ActiveSiteContentID        string            `json:"activeSiteContentId"`
	LatestSiteContentIDsByYear map[string]string `json:"latestSiteContentIdsByYear"`
	CSU                        bool              `json:"csu"`
	UC                         bool              `json:"uc"`
}

type pmSiteContent struct {
	ID              string `json:"id"`
	CollegeID       string `json:"collegeId"`
	CollegeName     string `json:"collegeName"`
	Year            int    `json:"year"`
	ProgramMapTitle string `json:"programMapTitle"`
}

// pmProgramSummary is a program as listed inside a program-group (cluster).
type pmProgramSummary struct {
	MasterRecordID    string `json:"masterRecordId"`
	Title             string `json:"title"`
	AwardShortTitle   string `json:"awardShortTitle"`
	AwardCategory     string `json:"awardCategory"`
	TransferAvailable bool   `json:"transferAvailable"`
}

// pmCluster is a program-group (Guided Pathways "Area of Interest").
type pmCluster struct {
	MasterRecordID string             `json:"masterRecordId"`
	Title          string             `json:"title"`
	Description    string             `json:"description"`
	Icon           string             `json:"icon"`
	Programs       []pmProgramSummary `json:"programs"`
}

// pmHomePage is the home-page-content envelope listing the clusters.
type pmHomePage struct {
	SiteContentID string      `json:"siteContentId"`
	Title         string      `json:"title"`
	About         string      `json:"about"`
	ProgramGroups []pmCluster `json:"programGroups"`
}

type pmPathway struct {
	ProgramMapID   string          `json:"programMapId"`
	Label          string          `json:"label"`
	DefaultPathway bool            `json:"defaultPathway"`
	Next           json.RawMessage `json:"next"`
}

// hasNext reports whether the pathway has a (non-null) transfer continuation.
func (p pmPathway) hasNext() bool {
	t := strings.TrimSpace(string(p.Next))
	return t != "" && t != "null"
}

// pmProgram is a program detail record.
type pmProgram struct {
	ProgramID     string      `json:"programId"`
	Title         string      `json:"title"`
	AwardTitle    string      `json:"awardTitle"`
	Description   string      `json:"description"`
	SiteContentID string      `json:"siteContentId"`
	CollegeID     string      `json:"collegeId"`
	CareerSOC     []string    `json:"careerStandardOccupationalCodes"`
	LinkedPathway bool        `json:"linkedPathway"`
	CatalogURL    string      `json:"catalogUrl"`
	Pathways      []pmPathway `json:"pathways"`
}

// defaultMapID returns the program's default pathway map id, or the first
// pathway's map id when none is flagged default.
func (p pmProgram) defaultMapID() string {
	for _, pw := range p.Pathways {
		if pw.DefaultPathway && pw.ProgramMapID != "" {
			return pw.ProgramMapID
		}
	}
	for _, pw := range p.Pathways {
		if pw.ProgramMapID != "" {
			return pw.ProgramMapID
		}
	}
	return ""
}

type pmTerm struct {
	CustomLabel string `json:"customLabel"`
	TermNumber  int    `json:"termNumber"`
	Year        int    `json:"year"`
}

type pmOpportunity struct {
	Type           string  `json:"type"` // COURSE | CHOICE | MILESTONE
	Term           pmTerm  `json:"term"`
	CourseCode     string  `json:"courseCode"`
	CourseName     string  `json:"courseName"`
	MappedCourseID string  `json:"mappedCourseId"`
	MinUnits       float64 `json:"minUnits"`
	MaxUnits       float64 `json:"maxUnits"`
}

type pmRequirement struct {
	RequirementType string `json:"requirementType"`
}

type pmPathwayElement struct {
	ID                     string        `json:"id"`
	Name                   string        `json:"name"`
	ShortDescription       string        `json:"shortDescription"`
	Requirement            pmRequirement `json:"requirement"`
	RecommendedOpportunity pmOpportunity `json:"recommendedOpportunity"`
}

type pmMap struct {
	ProgramMapID      string             `json:"programMapId"`
	SiteContentID     string             `json:"siteContentId"`
	Label             string             `json:"label"`
	TermsToCompletion int                `json:"termsToCompletion"`
	PathwayElements   []pmPathwayElement `json:"pathwayElements"`
	// Injected by mirror so course-programs/bottlenecks can link a map back to
	// its program without relying on the API's mislabeled map.programId field.
	PMProgramID    string `json:"pmProgramId,omitempty"`
	PMProgramTitle string `json:"pmProgramTitle,omitempty"`
	PMAward        string `json:"pmAward,omitempty"`
}

// ---- Helpers ----

// pmIsRateLimited reports whether an API error is a WAF rate-limit/block
// (403 or 429) rather than a genuine not-found or other failure. The
// ProgramMapper WAF throttles bursts per IP; empty-on-throttle would be
// indistinguishable from "no data exists", so callers surface a typed
// rate-limit error instead.
func pmIsRateLimited(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 429 || apiErr.StatusCode == 403
	}
	// Fallback for errors that don't wrap a typed APIError.
	msg := err.Error()
	return strings.Contains(msg, "HTTP 429") || strings.Contains(msg, "HTTP 403")
}

// pmResolveSiteContent resolves a user-supplied college reference (a full
// ProgramMapper vanity URL, or a college registry id like "la_mission") to its
// active site-content id, returning the college record for prior-year lookups.
func pmResolveSiteContent(ctx context.Context, c *client.Client, ref string) (string, pmCollege, error) {
	ref = strings.TrimSpace(ref)
	var raw json.RawMessage
	var err error
	if strings.Contains(ref, "://") {
		raw, err = c.Get(ctx, "/colleges", map[string]string{"url": ref})
		if err != nil {
			return "", pmCollege{}, err
		}
		var list []pmCollege
		if jerr := json.Unmarshal(raw, &list); jerr != nil {
			return "", pmCollege{}, fmt.Errorf("parsing colleges response: %w", jerr)
		}
		if len(list) == 0 {
			return "", pmCollege{}, notFoundErr(fmt.Errorf("no college found for URL %q (include the scheme, e.g. https://la-mission.programmapper.ws)", ref))
		}
		col := list[0]
		if col.ActiveSiteContentID == "" {
			return "", col, notFoundErr(fmt.Errorf("college %q has no active site-content", col.ID))
		}
		return col.ActiveSiteContentID, col, nil
	}
	raw, err = c.Get(ctx, "/colleges/"+ref, nil)
	if err != nil {
		return "", pmCollege{}, err
	}
	var col pmCollege
	if jerr := json.Unmarshal(raw, &col); jerr != nil {
		return "", pmCollege{}, fmt.Errorf("parsing college response: %w", jerr)
	}
	if col.ActiveSiteContentID == "" {
		return "", col, notFoundErr(fmt.Errorf("college %q has no active site-content", ref))
	}
	return col.ActiveSiteContentID, col, nil
}

// pmOpenStore opens the local SQLite store at the resolved db path. resolveDB
// returns the path used so callers can show it in hints.
func pmOpenStore(dbPath string) (*store.Store, error) {
	return store.Open(dbPath)
}

// pmDBPath returns the db path from --db or the default location.
func pmDBPath(explicit string) string {
	if explicit != "" {
		return explicit
	}
	return defaultDBPath("programmapper-cli")
}

// pmRequireMirror writes a "no local mirror" hint and reports whether the db
// is missing. Store-reading transcendence commands call this after dryRunOK
// and after resolving dbPath, before opening the store.
func pmRequireMirror(cmd *cobra.Command, dbPath string, flags *rootFlags) bool {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Fprintf(cmd.ErrOrStderr(), "no local mirror at %s\nrun: programmapper-cli mirror <college> --maps\n", dbPath)
		if flags.asJSON || flags.agent {
			fmt.Fprintln(cmd.OutOrStdout(), "[]")
		}
		return false
	}
	return true
}

// pmGetProgram loads a program detail from the store by id.
func pmGetProgram(db *store.Store, programID string) (pmProgram, bool, error) {
	raw, err := db.Get(rtPrograms, programID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return pmProgram{}, false, nil
		}
		return pmProgram{}, false, fmt.Errorf("reading program %s from store: %w", programID, err)
	}
	if len(raw) == 0 {
		return pmProgram{}, false, nil
	}
	var p pmProgram
	if jerr := json.Unmarshal(raw, &p); jerr != nil {
		return pmProgram{}, false, fmt.Errorf("parsing stored program %s: %w", programID, jerr)
	}
	return p, true, nil
}

// pmGetMap loads a program-map from the store by id.
func pmGetMap(db *store.Store, mapID string) (pmMap, bool, error) {
	raw, err := db.Get(rtProgramMaps, mapID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return pmMap{}, false, nil
		}
		return pmMap{}, false, fmt.Errorf("reading map %s from store: %w", mapID, err)
	}
	if len(raw) == 0 {
		return pmMap{}, false, nil
	}
	var m pmMap
	if jerr := json.Unmarshal(raw, &m); jerr != nil {
		return pmMap{}, false, fmt.Errorf("parsing stored map %s: %w", mapID, jerr)
	}
	return m, true, nil
}

// pmFetchMap fetches a program-map live from the API.
func pmFetchMap(ctx context.Context, c *client.Client, scid, mapID string) (pmMap, error) {
	raw, err := c.Get(ctx, fmt.Sprintf("/site-contents/%s/program-maps/%s", scid, mapID), nil)
	if err != nil {
		return pmMap{}, err
	}
	var m pmMap
	if jerr := json.Unmarshal(raw, &m); jerr != nil {
		return pmMap{}, fmt.Errorf("parsing map %s: %w", mapID, jerr)
	}
	return m, nil
}

// pmFetchProgram fetches a program detail live from the API.
func pmFetchProgram(ctx context.Context, c *client.Client, scid, programID string) (pmProgram, error) {
	raw, err := c.Get(ctx, fmt.Sprintf("/site-contents/%s/programs/%s", scid, programID), nil)
	if err != nil {
		return pmProgram{}, err
	}
	var p pmProgram
	if jerr := json.Unmarshal(raw, &p); jerr != nil {
		return pmProgram{}, fmt.Errorf("parsing program %s: %w", programID, jerr)
	}
	return p, nil
}

// pmLoadProgramMap loads a program and its default map, preferring the local
// store and falling back to a live fetch unless local is true. newClient is a
// lazy constructor so no client is built when the store already holds the data.
// Returns an empty pmMap (with ProgramMapID == "") when the map is unavailable
// in local-only mode.
func pmLoadProgramMap(ctx context.Context, newClient func() (*client.Client, error), db *store.Store, programID, scidOverride string, local bool) (pmProgram, pmMap, string, error) {
	prog, _, err := pmGetProgram(db, programID)
	if err != nil {
		return pmProgram{}, pmMap{}, "", err
	}
	scid := prog.SiteContentID
	if scidOverride != "" {
		scid = scidOverride
	}
	mapID := prog.defaultMapID()
	source := "local"
	if mapID == "" {
		if local {
			return prog, pmMap{}, "local", nil
		}
		if scid == "" {
			return prog, pmMap{}, "", usageErr(fmt.Errorf("program %s is not mirrored; pass --site-content or run: programmapper-cli mirror <college> --maps", programID))
		}
		c, cerr := newClient()
		if cerr != nil {
			return prog, pmMap{}, "", cerr
		}
		live, ferr := pmFetchProgram(ctx, c, scid, programID)
		if ferr != nil {
			return prog, pmMap{}, "", ferr
		}
		prog = live
		mapID = prog.defaultMapID()
		source = "live"
	}
	if mapID == "" {
		return prog, pmMap{}, source, nil
	}
	m, ok, err := pmGetMap(db, mapID)
	if err != nil {
		return prog, pmMap{}, "", err
	}
	if !ok {
		if local {
			return prog, pmMap{}, "local", nil
		}
		if scid == "" {
			scid = m.SiteContentID
		}
		if scid == "" {
			return prog, pmMap{}, "", usageErr(fmt.Errorf("cannot fetch map %s live without a site-content id; pass --site-content", mapID))
		}
		c, cerr := newClient()
		if cerr != nil {
			return prog, pmMap{}, "", cerr
		}
		live, ferr := pmFetchMap(ctx, c, scid, mapID)
		if ferr != nil {
			return prog, pmMap{}, "", ferr
		}
		m = live
		source = "live"
	}
	return prog, m, source, nil
}

// pmMapCourses returns the COURSE opportunities in a map keyed by course code,
// with units and name. CHOICE/MILESTONE elements are skipped.
func pmMapCourses(m pmMap) map[string]pmOpportunity {
	out := map[string]pmOpportunity{}
	for _, el := range m.PathwayElements {
		op := el.RecommendedOpportunity
		if op.Type != "COURSE" || op.CourseCode == "" {
			continue
		}
		out[op.CourseCode] = op
	}
	return out
}

// recordMirrorSyncState marks the mirrored resource types as synced so the
// store's "not synced yet" hints stop firing after a successful mirror.
func recordMirrorSyncState(db *store.Store, s *mirrorStats) {
	if s.siteContents > 0 {
		_ = db.SaveSyncState(rtSiteContents, "", s.siteContents)
	}
	if s.clusters > 0 {
		_ = db.SaveSyncState(rtClusters, "", s.clusters)
	}
	if s.programs > 0 {
		_ = db.SaveSyncState(rtPrograms, "", s.programs)
	}
	if s.maps > 0 {
		_ = db.SaveSyncState(rtProgramMaps, "", s.maps)
	}
	if s.courses > 0 {
		_ = db.SaveSyncState(rtCourses, "", s.courses)
	}
}

// pmUnitsLabel renders a min/max units pair as "3" or "1-4".
func pmUnitsLabel(min, max float64) string {
	if min == max {
		return trimFloat(min)
	}
	return trimFloat(min) + "-" + trimFloat(max)
}

func trimFloat(f float64) string {
	if f == float64(int64(f)) {
		return fmt.Sprintf("%d", int64(f))
	}
	return fmt.Sprintf("%.1f", f)
}
