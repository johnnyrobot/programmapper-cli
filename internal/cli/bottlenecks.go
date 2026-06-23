// Copyright 2026 johnnyrobot and contributors. Licensed under Apache-2.0. See LICENSE.
//
// bottlenecks.go — hand-authored transcendence command. Ranks courses across the
// mirrored college by how many distinct program maps require them. ProgramMapper
// exposes no aggregation endpoint, so this is computed over the local mirror.
//
// pp:data-source local

package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

type bottleneckRow struct {
	CourseCode   string  `json:"course_code"`
	CourseName   string  `json:"course_name,omitempty"`
	ProgramCount int     `json:"program_count"`
	MinUnits     float64 `json:"min_units"`
	MaxUnits     float64 `json:"max_units"`
}

type bottlenecksView struct {
	Courses     []bottleneckRow `json:"courses"`
	ScannedMaps int             `json:"scanned_maps"`
	Note        string          `json:"note,omitempty"`
}

func newNovelBottlenecksCmd(flags *rootFlags) *cobra.Command {
	var dbPath string
	var limit int

	cmd := &cobra.Command{
		Use:   "bottlenecks",
		Short: "Rank courses across the mirrored college by how many program maps require them",
		Long: strings.TrimSpace(`
Rank courses by how many distinct program maps require them, surfacing the
high-leverage courses that unlock the most programs. Requires a deep mirror
(programmapper-cli mirror <college> --maps).`),
		Example:     "  programmapper-cli bottlenecks --limit 20 --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			if flags.dataSource == "live" {
				return usageErr(fmt.Errorf("bottlenecks has no live equivalent; it aggregates the local mirror (run: mirror <college> --maps)"))
			}

			dbPath = pmDBPath(dbPath)
			if !pmRequireMirror(cmd, dbPath, flags) {
				return nil
			}
			db, err := pmOpenStore(dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()
			hintIfUnsynced(cmd, db, rtProgramMaps)

			rows, err := db.List(rtProgramMaps, 0)
			if err != nil {
				return fmt.Errorf("listing maps: %w", err)
			}
			type agg struct {
				name     string
				programs map[string]bool
				minUnits float64
				maxUnits float64
				unitsSet bool
			}
			byCode := map[string]*agg{}
			for _, raw := range rows {
				var m pmMap
				if json.Unmarshal(raw, &m) != nil {
					continue
				}
				progKey := firstNonEmpty(m.PMProgramID, m.ProgramMapID)
				seenInMap := map[string]bool{}
				for _, el := range m.PathwayElements {
					op := el.RecommendedOpportunity
					if op.Type != "COURSE" || op.CourseCode == "" || seenInMap[op.CourseCode] {
						continue
					}
					seenInMap[op.CourseCode] = true
					a := byCode[op.CourseCode]
					if a == nil {
						a = &agg{name: op.CourseName, programs: map[string]bool{}}
						byCode[op.CourseCode] = a
					}
					a.programs[progKey] = true
					// Track the min/max units seen across every map that uses
					// this course, rather than last-writer-wins. Zero-unit
					// courses (labs, milestones-as-course) are legitimate and
					// not dropped.
					if !a.unitsSet {
						a.minUnits = op.MinUnits
						a.maxUnits = op.MaxUnits
						a.unitsSet = true
					} else {
						if op.MinUnits < a.minUnits {
							a.minUnits = op.MinUnits
						}
						if op.MaxUnits > a.maxUnits {
							a.maxUnits = op.MaxUnits
						}
					}
					if a.name == "" {
						a.name = op.CourseName
					}
				}
			}

			view := bottlenecksView{ScannedMaps: len(rows), Courses: []bottleneckRow{}}
			for code, a := range byCode {
				view.Courses = append(view.Courses, bottleneckRow{
					CourseCode:   code,
					CourseName:   a.name,
					ProgramCount: len(a.programs),
					MinUnits:     a.minUnits,
					MaxUnits:     a.maxUnits,
				})
			}
			sort.Slice(view.Courses, func(i, j int) bool {
				if view.Courses[i].ProgramCount != view.Courses[j].ProgramCount {
					return view.Courses[i].ProgramCount > view.Courses[j].ProgramCount
				}
				return view.Courses[i].CourseCode < view.Courses[j].CourseCode
			})
			if limit > 0 && len(view.Courses) > limit {
				view.Courses = view.Courses[:limit]
			}
			if view.ScannedMaps == 0 {
				view.Note = "no maps in the local mirror; run: programmapper-cli mirror <college> --maps"
			}

			if flags.asJSON || (!isTerminal(cmd.OutOrStdout()) && !flags.csv && !flags.quiet && !flags.plain) {
				return printJSONFiltered(cmd.OutOrStdout(), view, flags)
			}
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "%-16s %-8s %s\n", bold("COURSE"), bold("PROGRAMS"), bold("NAME"))
			for _, c := range view.Courses {
				fmt.Fprintf(w, "%-16s %-8d %s\n", c.CourseCode, c.ProgramCount, c.CourseName)
			}
			if view.Note != "" {
				fmt.Fprintf(w, "note: %s\n", view.Note)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum courses to return")
	return cmd
}
