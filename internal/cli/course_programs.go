// Copyright 2026 johnnyrobot and contributors. Licensed under Apache-2.0. See LICENSE.
//
// course_programs.go — hand-authored transcendence command. Reverse lookup:
// every program at the mirrored college whose term plan includes a given course.
// ProgramMapper only navigates program->map->course, never the reverse, so this
// requires a local index over the mirrored maps. Reads the local store only.
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

type courseProgramHit struct {
	ProgramID    string `json:"program_id"`
	ProgramTitle string `json:"program_title"`
	Award        string `json:"award,omitempty"`
	MapID        string `json:"program_map_id"`
	CourseCode   string `json:"course_code"`
	CourseName   string `json:"course_name,omitempty"`
	TermNumber   int    `json:"term_number"`
}

type courseProgramsView struct {
	Query       string             `json:"query"`
	Programs    []courseProgramHit `json:"programs"`
	ScannedMaps int                `json:"scanned_maps"`
	Note        string             `json:"note,omitempty"`
}

func newNovelCourseProgramsCmd(flags *rootFlags) *cobra.Command {
	var dbPath string

	cmd := &cobra.Command{
		Use:   "course-programs <courseIdOrCode>",
		Short: "List every program at the mirrored college whose plan includes a given course",
		Long: strings.TrimSpace(`
Reverse lookup: list every program whose term plan includes a course. Accepts a
course id (mappedCourseId) or a course code (e.g. "NURSING 090").

Use this to find which programs REQUIRE a course. To fetch a single course's
units and description instead, use 'courses get'. Requires a deep mirror
(programmapper-cli mirror <college> --maps).`),
		Example: "  programmapper-cli course-programs \"NURSING 090\" --json",
		// no-error-path-probe: any course id or code is valid input; a
		// non-matching value returns an honest empty result, not an error.
		Annotations: map[string]string{"mcp:read-only": "true", "pp:no-error-path-probe": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("a course id or course code is required"))
			}
			if flags.dataSource == "live" {
				return usageErr(fmt.Errorf("course-programs has no live equivalent; it reverse-indexes the local mirror (run: mirror <college> --maps)"))
			}
			query := strings.TrimSpace(args[0])
			queryLower := strings.ToLower(query)

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
			view := courseProgramsView{Query: query, ScannedMaps: len(rows), Programs: []courseProgramHit{}}
			seen := map[string]bool{}
			for _, raw := range rows {
				var m pmMap
				if json.Unmarshal(raw, &m) != nil {
					continue
				}
				for _, el := range m.PathwayElements {
					op := el.RecommendedOpportunity
					if op.Type != "COURSE" {
						continue
					}
					if op.MappedCourseID != query && !strings.EqualFold(op.CourseCode, query) && !strings.Contains(strings.ToLower(op.CourseCode), queryLower) {
						continue
					}
					key := m.PMProgramID + "|" + m.ProgramMapID
					if seen[key] {
						continue
					}
					seen[key] = true
					view.Programs = append(view.Programs, courseProgramHit{
						ProgramID:    m.PMProgramID,
						ProgramTitle: m.PMProgramTitle,
						Award:        m.PMAward,
						MapID:        m.ProgramMapID,
						CourseCode:   op.CourseCode,
						CourseName:   op.CourseName,
						TermNumber:   op.Term.TermNumber,
					})
					break
				}
			}
			sort.Slice(view.Programs, func(i, j int) bool {
				return view.Programs[i].ProgramTitle < view.Programs[j].ProgramTitle
			})
			if len(view.Programs) == 0 {
				if view.ScannedMaps == 0 {
					view.Note = "no maps in the local mirror; run: programmapper-cli mirror <college> --maps"
				} else {
					view.Note = fmt.Sprintf("scanned %d mirrored maps; no program uses %q", view.ScannedMaps, query)
				}
			}

			if flags.asJSON || (!isTerminal(cmd.OutOrStdout()) && !flags.csv && !flags.quiet && !flags.plain) {
				return printJSONFiltered(cmd.OutOrStdout(), view, flags)
			}
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "%d program(s) include %q (scanned %d maps)\n", len(view.Programs), query, view.ScannedMaps)
			for _, p := range view.Programs {
				fmt.Fprintf(w, "  %-40s %s  (term %d, %s)\n", p.ProgramTitle, p.Award, p.TermNumber, p.CourseCode)
			}
			if view.Note != "" {
				fmt.Fprintf(w, "note: %s\n", view.Note)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	return cmd
}
