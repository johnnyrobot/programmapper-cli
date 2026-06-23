// Copyright 2026 johnnyrobot and contributors. Licensed under Apache-2.0. See LICENSE.
//
// diff_years.go — hand-authored transcendence command. Diffs a program's current
// map against a prior catalog year (2019-2025): added/removed/changed courses and
// unit deltas. ProgramMapper has no diff view. The prior-year map is always
// fetched live (it is not part of the active-year mirror); the current map is read
// from the local mirror when present.
//
// pp:data-source auto

package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

type pmPriorYearEntry struct {
	Year           int    `json:"year"`
	ID             string `json:"id"`
	PathwaySummary struct {
		SiteContentID string `json:"siteContentId"`
		ProgramID     string `json:"programId"`
		ProgramMapID  string `json:"programMapId"`
		Label         string `json:"label"`
	} `json:"pathwaySummary"`
}

type diffCourse struct {
	Code     string  `json:"code"`
	Name     string  `json:"name,omitempty"`
	MinUnits float64 `json:"min_units"`
	MaxUnits float64 `json:"max_units"`
}

type diffYearsView struct {
	ProgramID    string       `json:"program_id"`
	Title        string       `json:"title"`
	CurrentMapID string       `json:"current_map_id"`
	PriorYear    int          `json:"prior_year"`
	PriorMapID   string       `json:"prior_map_id"`
	Added        []diffCourse `json:"added"`
	Removed      []diffCourse `json:"removed"`
	UnitDelta    float64      `json:"unit_delta_min"`
	Note         string       `json:"note,omitempty"`
}

func newNovelDiffYearsCmd(flags *rootFlags) *cobra.Command {
	var dbPath string
	var fromYear int
	var siteContent string

	cmd := &cobra.Command{
		Use:   "diff-years <programId>",
		Short: "Diff a program's current map against a prior catalog year (added/removed courses, unit deltas)",
		Long: strings.TrimSpace(`
Diff a program's current map against the same program's map in a prior catalog
year, reporting added and removed courses and the units delta. The prior-year map
is fetched live; pass --from-year to pick the year (default: the most recent prior
year that has a map).`),
		Example:     "  programmapper-pp-cli diff-years 4a0cb2c2-b22f-324d-834e-80cbc2bde5f4 --from-year 2023 --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("a program id is required"))
			}
			if flags.dataSource == "local" {
				return usageErr(fmt.Errorf("diff-years needs live access to fetch the prior catalog year; --data-source local is not supported"))
			}
			programID := args[0]

			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			dbPath = pmDBPath(dbPath)
			if !pmRequireMirror(cmd, dbPath, flags) {
				return nil
			}
			db, err := pmOpenStore(dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()
			hintIfUnsynced(cmd, db, rtPrograms)

			// Current side: program + map (store first, live fallback).
			prog, curMap, _, err := pmLoadProgramMap(ctx, flags.newClient, db, programID, siteContent, false)
			if err != nil {
				if pmIsRateLimited(err) {
					return rateLimitErr(err)
				}
				return classifyAPIError(err, flags)
			}
			if curMap.ProgramMapID == "" {
				return notFoundErr(fmt.Errorf("no current map found for program %s (try 'mirror --maps' or --site-content)", programID))
			}
			scid := prog.SiteContentID
			if siteContent != "" {
				scid = siteContent
			}
			if scid == "" {
				scid = curMap.SiteContentID
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			// Prior-years-with-maps (live).
			pyRaw, err := c.Get(ctx, fmt.Sprintf("/site-contents/%s/programs/%s/prior-years-with-maps", scid, programID), nil)
			if err != nil {
				if pmIsRateLimited(err) {
					return rateLimitErr(err)
				}
				return classifyAPIError(err, flags)
			}
			var priorYears []pmPriorYearEntry
			if jerr := json.Unmarshal(pyRaw, &priorYears); jerr != nil {
				return fmt.Errorf("parsing prior-years-with-maps: %w", jerr)
			}
			if len(priorYears) == 0 {
				return notFoundErr(fmt.Errorf("program %s has no prior catalog years with maps", programID))
			}
			// Newest prior year first, regardless of the API's ordering, so the
			// default (--from-year 0) is the most recent prior catalog.
			sort.Slice(priorYears, func(i, j int) bool { return priorYears[i].Year > priorYears[j].Year })
			var chosen *pmPriorYearEntry
			if fromYear == 0 {
				chosen = &priorYears[0]
			} else {
				for i := range priorYears {
					if priorYears[i].Year == fromYear {
						chosen = &priorYears[i]
						break
					}
				}
			}
			if chosen == nil {
				avail := make([]string, 0, len(priorYears))
				for _, p := range priorYears {
					avail = append(avail, fmt.Sprintf("%d", p.Year))
				}
				return usageErr(fmt.Errorf("no prior map for year %d; available: %s", fromYear, strings.Join(avail, ", ")))
			}

			priorScid := firstNonEmpty(chosen.PathwaySummary.SiteContentID, chosen.ID)
			priorMapID := chosen.PathwaySummary.ProgramMapID
			if priorScid == "" || priorMapID == "" {
				return notFoundErr(fmt.Errorf("prior year %d has no usable map reference", chosen.Year))
			}
			priorMap, err := pmFetchMap(ctx, c, priorScid, priorMapID)
			if err != nil {
				if pmIsRateLimited(err) {
					return rateLimitErr(err)
				}
				return classifyAPIError(err, flags)
			}

			view := buildDiffYears(prog, curMap, priorMap, chosen.Year)

			if flags.asJSON || (!isTerminal(cmd.OutOrStdout()) && !flags.csv && !flags.quiet && !flags.plain) {
				return printJSONFiltered(cmd.OutOrStdout(), view, flags)
			}
			renderDiffYears(cmd, view)
			return nil
		},
	}
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	cmd.Flags().IntVar(&fromYear, "from-year", 0, "Prior catalog year to diff against (default: most recent prior year)")
	cmd.Flags().StringVar(&siteContent, "site-content", "", "Site-content id (for live fetch when the program is not mirrored)")
	return cmd
}

func buildDiffYears(prog pmProgram, cur, prior pmMap, priorYear int) diffYearsView {
	curC := pmMapCourses(cur)
	priorC := pmMapCourses(prior)
	view := diffYearsView{
		ProgramID:    firstNonEmpty(prog.ProgramID, cur.PMProgramID),
		Title:        firstNonEmpty(prog.Title, cur.PMProgramTitle),
		CurrentMapID: cur.ProgramMapID,
		PriorYear:    priorYear,
		PriorMapID:   prior.ProgramMapID,
		Added:        []diffCourse{},
		Removed:      []diffCourse{},
	}
	var curUnits, priorUnits float64
	for code, op := range curC {
		curUnits += op.MinUnits
		if _, ok := priorC[code]; !ok {
			view.Added = append(view.Added, diffCourse{Code: code, Name: op.CourseName, MinUnits: op.MinUnits, MaxUnits: op.MaxUnits})
		}
	}
	for code, op := range priorC {
		priorUnits += op.MinUnits
		if _, ok := curC[code]; !ok {
			view.Removed = append(view.Removed, diffCourse{Code: code, Name: op.CourseName, MinUnits: op.MinUnits, MaxUnits: op.MaxUnits})
		}
	}
	sort.Slice(view.Added, func(i, j int) bool { return view.Added[i].Code < view.Added[j].Code })
	sort.Slice(view.Removed, func(i, j int) bool { return view.Removed[i].Code < view.Removed[j].Code })
	view.UnitDelta = curUnits - priorUnits
	if len(view.Added) == 0 && len(view.Removed) == 0 {
		view.Note = fmt.Sprintf("no course changes between %d and the current catalog", priorYear)
	}
	return view
}

func renderDiffYears(cmd *cobra.Command, view diffYearsView) {
	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "%s — changes since %d catalog (min-unit delta: %+.1f)\n\n", bold(view.Title), view.PriorYear, view.UnitDelta)
	fmt.Fprintf(w, "%s (%d):\n", bold("Added"), len(view.Added))
	for _, c := range view.Added {
		fmt.Fprintf(w, "  + %-14s %s\n", c.Code, c.Name)
	}
	fmt.Fprintf(w, "\n%s (%d):\n", bold("Removed"), len(view.Removed))
	for _, c := range view.Removed {
		fmt.Fprintf(w, "  - %-14s %s\n", c.Code, c.Name)
	}
	if view.Note != "" {
		fmt.Fprintf(w, "\n%s\n", view.Note)
	}
}
