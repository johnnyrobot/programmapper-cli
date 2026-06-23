// Copyright 2026 johnnyrobot and contributors. Licensed under Apache-2.0. See LICENSE.
//
// plan.go — hand-authored transcendence command. Builds a program's full
// term-by-term plan with per-term and cumulative units from the program map's
// pathway elements. Reads the local mirror first and falls back to a live fetch
// (data-source auto); no single ProgramMapper API call returns a units-totaled
// plan.
//
// pp:data-source auto

package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

type planItemView struct {
	Type        string  `json:"type"`
	Code        string  `json:"code,omitempty"`
	Name        string  `json:"name"`
	CourseID    string  `json:"course_id,omitempty"`
	Requirement string  `json:"requirement,omitempty"`
	MinUnits    float64 `json:"min_units"`
	MaxUnits    float64 `json:"max_units"`
}

type planTermView struct {
	TermNumber int            `json:"term_number"`
	Label      string         `json:"label,omitempty"`
	Year       int            `json:"year,omitempty"`
	Items      []planItemView `json:"items"`
	MinUnits   float64        `json:"term_min_units"`
	MaxUnits   float64        `json:"term_max_units"`
}

type planView struct {
	ProgramID         string         `json:"program_id"`
	Title             string         `json:"title"`
	Award             string         `json:"award,omitempty"`
	MapID             string         `json:"program_map_id"`
	TermsToCompletion int            `json:"terms_to_completion,omitempty"`
	Terms             []planTermView `json:"terms"`
	TotalMinUnits     float64        `json:"total_min_units"`
	TotalMaxUnits     float64        `json:"total_max_units"`
	Source            string         `json:"source"`
}

func newNovelPlanCmd(flags *rootFlags) *cobra.Command {
	var dbPath string
	var siteContent string

	cmd := &cobra.Command{
		Use:   "plan <programId>",
		Short: "Build a program's full semester-by-semester plan with per-term and cumulative units",
		Long: strings.TrimSpace(`
Build a program's full term-by-term plan with per-term and cumulative units,
expanding required courses, "choose one of" options, and milestones.

Use this for the planned course sequence WITH units rollups. To fetch the raw
map JSON for a single map id instead, use 'program-maps get'.

Reads the local mirror first; if the program or map is not mirrored it fetches
live (pass --site-content when the program is not in the local store).`),
		Example:     "  programmapper-pp-cli plan 4a0cb2c2-b22f-324d-834e-80cbc2bde5f4 --agent",
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
			programID := args[0]
			local := flags.dataSource == "local"

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

			prog, _, err := pmGetProgram(db, programID)
			if err != nil {
				return err
			}
			scid := prog.SiteContentID
			if siteContent != "" {
				scid = siteContent
			}

			mapID := prog.defaultMapID()
			source := "local"

			// Enrich from live when the local record is a bare summary (no map id).
			if mapID == "" && !local {
				if scid == "" {
					return usageErr(fmt.Errorf("program %s is not deeply mirrored; pass --site-content <id> or run: programmapper-pp-cli mirror <college> --maps", programID))
				}
				c, cerr := flags.newClient()
				if cerr != nil {
					return cerr
				}
				live, ferr := pmFetchProgram(ctx, c, scid, programID)
				if ferr != nil {
					if pmIsRateLimited(ferr) {
						return rateLimitErr(ferr)
					}
					return classifyAPIError(ferr, flags)
				}
				prog = live
				mapID = prog.defaultMapID()
				source = "live"
			}
			if mapID == "" {
				return notFoundErr(fmt.Errorf("no program map found for program %s (run 'mirror --maps' or check the id)", programID))
			}

			m, ok, err := pmGetMap(db, mapID)
			if err != nil {
				return err
			}
			if !ok {
				if local {
					fmt.Fprintf(cmd.ErrOrStderr(), "map %s not in local mirror; run: programmapper-pp-cli mirror <college> --maps\n", mapID)
					if flags.asJSON || flags.agent {
						fmt.Fprintln(cmd.OutOrStdout(), "[]")
					}
					return nil
				}
				if scid == "" {
					scid = m.SiteContentID
				}
				if scid == "" {
					return usageErr(fmt.Errorf("cannot fetch map %s live without a site-content id; pass --site-content", mapID))
				}
				c, cerr := flags.newClient()
				if cerr != nil {
					return cerr
				}
				live, ferr := pmFetchMap(ctx, c, scid, mapID)
				if ferr != nil {
					if pmIsRateLimited(ferr) {
						return rateLimitErr(ferr)
					}
					return classifyAPIError(ferr, flags)
				}
				m = live
				source = "live"
			}

			view := buildPlan(prog, m)
			view.Source = source

			if flags.asJSON || (!isTerminal(cmd.OutOrStdout()) && !flags.csv && !flags.quiet && !flags.plain) {
				return printJSONFiltered(cmd.OutOrStdout(), view, flags)
			}
			renderPlan(cmd, view)
			return nil
		},
	}
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	cmd.Flags().StringVar(&siteContent, "site-content", "", "Site-content id (required for live fetch when the program is not mirrored)")
	return cmd
}

func buildPlan(prog pmProgram, m pmMap) planView {
	title := prog.Title
	if title == "" {
		title = m.PMProgramTitle
	}
	award := prog.AwardTitle
	if award == "" {
		award = m.PMAward
	}
	view := planView{
		ProgramID:         prog.ProgramID,
		Title:             title,
		Award:             award,
		MapID:             m.ProgramMapID,
		TermsToCompletion: m.TermsToCompletion,
	}
	if view.ProgramID == "" {
		view.ProgramID = m.PMProgramID
	}

	termIdx := map[int]*planTermView{}
	var order []int
	for _, el := range m.PathwayElements {
		op := el.RecommendedOpportunity
		tn := op.Term.TermNumber
		t, ok := termIdx[tn]
		if !ok {
			t = &planTermView{TermNumber: tn, Label: strings.TrimSpace(op.Term.CustomLabel), Year: op.Term.Year}
			termIdx[tn] = t
			order = append(order, tn)
		}
		item := planItemView{
			Type:        op.Type,
			Code:        op.CourseCode,
			Name:        firstNonEmpty(op.CourseName, el.Name),
			CourseID:    op.MappedCourseID,
			Requirement: el.Requirement.RequirementType,
			MinUnits:    op.MinUnits,
			MaxUnits:    op.MaxUnits,
		}
		if item.Name == "" {
			item.Name = el.ShortDescription
		}
		t.Items = append(t.Items, item)
		t.MinUnits += op.MinUnits
		t.MaxUnits += op.MaxUnits
		view.TotalMinUnits += op.MinUnits
		view.TotalMaxUnits += op.MaxUnits
	}
	sort.Ints(order)
	for _, tn := range order {
		view.Terms = append(view.Terms, *termIdx[tn])
	}
	return view
}

func renderPlan(cmd *cobra.Command, view planView) {
	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "%s — %s\n", bold(view.Title), view.Award)
	fmt.Fprintf(w, "Total units: %s   (terms to completion: %d)\n\n", pmUnitsLabel(view.TotalMinUnits, view.TotalMaxUnits), view.TermsToCompletion)
	for _, t := range view.Terms {
		label := t.Label
		if label == "" {
			label = fmt.Sprintf("Term %d", t.TermNumber)
		}
		fmt.Fprintf(w, "%s (%s units)\n", bold(fmt.Sprintf("Term %d — %s %d", t.TermNumber, label, t.Year)), pmUnitsLabel(t.MinUnits, t.MaxUnits))
		for _, it := range t.Items {
			switch it.Type {
			case "COURSE":
				fmt.Fprintf(w, "  %-14s %s  (%s units)\n", it.Code, it.Name, pmUnitsLabel(it.MinUnits, it.MaxUnits))
			case "CHOICE":
				fmt.Fprintf(w, "  %-14s %s  (choose one, %s units)\n", "[choice]", it.Name, pmUnitsLabel(it.MinUnits, it.MaxUnits))
			default:
				fmt.Fprintf(w, "  %-14s %s\n", "["+strings.ToLower(it.Type)+"]", it.Name)
			}
		}
		fmt.Fprintln(w)
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
