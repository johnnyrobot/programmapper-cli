// Copyright 2026 johnnyrobot and contributors. Licensed under Apache-2.0. See LICENSE.
//
// compare.go — hand-authored transcendence command. Diffs two programs' course
// sets and unit totals side by side using the local mirror (live fallback). The
// ProgramMapper UI can only render one map at a time.
//
// pp:data-source auto

package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

type compareCourse struct {
	Code     string  `json:"code"`
	Name     string  `json:"name,omitempty"`
	MinUnits float64 `json:"min_units"`
	MaxUnits float64 `json:"max_units"`
}

type compareSide struct {
	ProgramID string  `json:"program_id"`
	Title     string  `json:"title"`
	Award     string  `json:"award,omitempty"`
	Courses   int     `json:"course_count"`
	MinUnits  float64 `json:"total_min_units"`
	MaxUnits  float64 `json:"total_max_units"`
}

type compareView struct {
	A           compareSide     `json:"a"`
	B           compareSide     `json:"b"`
	Shared      []compareCourse `json:"shared_courses"`
	OnlyA       []compareCourse `json:"only_a"`
	OnlyB       []compareCourse `json:"only_b"`
	SharedCount int             `json:"shared_count"`
	Source      string          `json:"source"`
}

func newNovelCompareCmd(flags *rootFlags) *cobra.Command {
	var dbPath string
	var siteContent string

	cmd := &cobra.Command{
		Use:         "compare <programIdA> <programIdB>",
		Short:       "Show two programs side by side: shared courses, unique courses, and per-program units totals",
		Example:     "  programmapper-pp-cli compare 4a0cb2c2-b22f-324d-834e-80cbc2bde5f4 a4060608-61af-8a69-5d00-66fc77c61774 --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if len(args) < 2 {
				if dryRunOK(flags) {
					return nil
				}
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("two program ids are required"))
			}
			if dryRunOK(flags) {
				return nil
			}
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

			progA, mapA, srcA, err := pmLoadProgramMap(ctx, flags.newClient, db, args[0], siteContent, local)
			if err != nil {
				if pmIsRateLimited(err) {
					return rateLimitErr(err)
				}
				return classifyAPIError(err, flags)
			}
			progB, mapB, srcB, err := pmLoadProgramMap(ctx, flags.newClient, db, args[1], siteContent, local)
			if err != nil {
				if pmIsRateLimited(err) {
					return rateLimitErr(err)
				}
				return classifyAPIError(err, flags)
			}
			if mapA.ProgramMapID == "" || mapB.ProgramMapID == "" {
				fmt.Fprintln(cmd.ErrOrStderr(), "one or both program maps are not available; run: programmapper-pp-cli mirror <college> --maps")
				if flags.asJSON || flags.agent {
					fmt.Fprintln(cmd.OutOrStdout(), "{}")
				}
				return nil
			}

			view := buildCompare(progA, mapA, progB, mapB)
			if srcA == "live" || srcB == "live" {
				view.Source = "live"
			} else {
				view.Source = "local"
			}

			if flags.asJSON || (!isTerminal(cmd.OutOrStdout()) && !flags.csv && !flags.quiet && !flags.plain) {
				return printJSONFiltered(cmd.OutOrStdout(), view, flags)
			}
			renderCompare(cmd, view)
			return nil
		},
	}
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	cmd.Flags().StringVar(&siteContent, "site-content", "", "Site-content id (for live fetch when a program is not mirrored)")
	return cmd
}

func buildCompare(progA pmProgram, mapA pmMap, progB pmProgram, mapB pmMap) compareView {
	ca := pmMapCourses(mapA)
	cb := pmMapCourses(mapB)
	side := func(prog pmProgram, m pmMap, courses map[string]pmOpportunity) compareSide {
		s := compareSide{
			ProgramID: firstNonEmpty(prog.ProgramID, m.PMProgramID),
			Title:     firstNonEmpty(prog.Title, m.PMProgramTitle),
			Award:     firstNonEmpty(prog.AwardTitle, m.PMAward),
			Courses:   len(courses),
		}
		for _, op := range courses {
			s.MinUnits += op.MinUnits
			s.MaxUnits += op.MaxUnits
		}
		return s
	}
	view := compareView{A: side(progA, mapA, ca), B: side(progB, mapB, cb)}

	toCourse := func(op pmOpportunity) compareCourse {
		return compareCourse{Code: op.CourseCode, Name: op.CourseName, MinUnits: op.MinUnits, MaxUnits: op.MaxUnits}
	}
	for code, op := range ca {
		if _, ok := cb[code]; ok {
			view.Shared = append(view.Shared, toCourse(op))
		} else {
			view.OnlyA = append(view.OnlyA, toCourse(op))
		}
	}
	for code, op := range cb {
		if _, ok := ca[code]; !ok {
			view.OnlyB = append(view.OnlyB, toCourse(op))
		}
	}
	sortCourses(view.Shared)
	sortCourses(view.OnlyA)
	sortCourses(view.OnlyB)
	view.SharedCount = len(view.Shared)
	return view
}

func sortCourses(c []compareCourse) {
	sort.Slice(c, func(i, j int) bool { return c[i].Code < c[j].Code })
}

func renderCompare(cmd *cobra.Command, view compareView) {
	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "%s\n  A: %s — %s (%d courses, %s units)\n  B: %s — %s (%d courses, %s units)\n\n",
		bold("Program comparison"),
		view.A.Title, view.A.Award, view.A.Courses, pmUnitsLabel(view.A.MinUnits, view.A.MaxUnits),
		view.B.Title, view.B.Award, view.B.Courses, pmUnitsLabel(view.B.MinUnits, view.B.MaxUnits))
	fmt.Fprintf(w, "%s (%d):\n", bold("Shared courses"), len(view.Shared))
	for _, c := range view.Shared {
		fmt.Fprintf(w, "  %-14s %s\n", c.Code, c.Name)
	}
	fmt.Fprintf(w, "\n%s (%d):\n", bold("Only in A — "+view.A.Title), len(view.OnlyA))
	for _, c := range view.OnlyA {
		fmt.Fprintf(w, "  %-14s %s\n", c.Code, c.Name)
	}
	fmt.Fprintf(w, "\n%s (%d):\n", bold("Only in B — "+view.B.Title), len(view.OnlyB))
	for _, c := range view.OnlyB {
		fmt.Fprintf(w, "  %-14s %s\n", c.Code, c.Name)
	}
}
