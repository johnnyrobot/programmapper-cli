// Copyright 2026 johnnyrobot and contributors. Licensed under Apache-2.0. See LICENSE.
//
// transfer_options.go — hand-authored transcendence command. Rolls up where a
// program leads: its transfer pathways (CSU/UC / Cal-GETC continuations) and the
// career outlook (salary + job-growth) for the program's mapped SOC codes. The
// ProgramMapper UI surfaces these one at a time; this joins them in one view.
//
// pp:data-source auto

package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type pmCareer struct {
	Title          string   `json:"title"`
	SOC            string   `json:"standardOccupationalCode"`
	AverageSalaryK *float64 `json:"averageSalaryK"`
	LowSalaryK     *float64 `json:"lowSalaryK"`
	HighSalaryK    *float64 `json:"highSalaryK"`
	JobGrowthPct   *float64 `json:"jobGrowthPct"`
	EducationLevel string   `json:"educationLevel"`
}

type pmCareersBatch struct {
	BatchLowSalaryK     *float64   `json:"batchLowSalaryK"`
	BatchAverageSalaryK *float64   `json:"batchAverageSalaryK"`
	BatchHighSalaryK    *float64   `json:"batchHighSalaryK"`
	BatchJobGrowthPct   *float64   `json:"batchJobGrowthPct"`
	Careers             []pmCareer `json:"careers"`
}

type transferPathwayView struct {
	Label                string `json:"label"`
	ProgramMapID         string `json:"program_map_id"`
	DefaultPathway       bool   `json:"default_pathway"`
	TransferContinuation bool   `json:"transfer_continuation"`
}

type careerOutlookView struct {
	AvgSalaryK   *float64   `json:"avg_salary_k,omitempty"`
	LowSalaryK   *float64   `json:"low_salary_k,omitempty"`
	HighSalaryK  *float64   `json:"high_salary_k,omitempty"`
	JobGrowthPct *float64   `json:"job_growth_pct,omitempty"`
	Careers      []pmCareer `json:"careers"`
}

type transferOptionsView struct {
	ProgramID          string                `json:"program_id"`
	Title              string                `json:"title"`
	Award              string                `json:"award,omitempty"`
	TransferDesignated bool                  `json:"transfer_designated"`
	LinkedPathway      bool                  `json:"linked_pathway"`
	Pathways           []transferPathwayView `json:"transfer_pathways"`
	CareerOutlook      *careerOutlookView    `json:"career_outlook,omitempty"`
	Source             string                `json:"source"`
	Note               string                `json:"note,omitempty"`
}

func newNovelTransferOptionsCmd(flags *rootFlags) *cobra.Command {
	var dbPath string
	var siteContent string

	cmd := &cobra.Command{
		Use:   "transfer-options <programId>",
		Short: "A program's transfer pathways (CSU/UC) and career outlook in one view",
		Long: strings.TrimSpace(`
Roll up where a program leads: its transfer pathways (CSU/UC, Cal-GETC
continuations) and the career outlook (salary range, job growth) for its mapped
SOC codes.

Use this for a program's combined transfer + career outlook. To fetch a single
linked transfer-college record by id instead, use 'transfer linked-college'.`),
		Example:     "  programmapper-cli transfer-options a4060608-61af-8a69-5d00-66fc77c61774 --json",
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
			source := "local"
			// A summary-only record has no pathways; enrich from live.
			if len(prog.Pathways) == 0 && !local {
				if scid == "" {
					return usageErr(fmt.Errorf("program %s is not deeply mirrored; pass --site-content or run mirror --maps", programID))
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
				source = "live"
			}
			if prog.ProgramID == "" && len(prog.Pathways) == 0 {
				return notFoundErr(fmt.Errorf("program %s not found (run 'mirror --maps' or check the id)", programID))
			}

			view := transferOptionsView{
				ProgramID:          firstNonEmpty(prog.ProgramID, programID),
				Title:              prog.Title,
				Award:              prog.AwardTitle,
				LinkedPathway:      prog.LinkedPathway,
				TransferDesignated: isTransferDesignated(prog),
				Source:             source,
				Pathways:           []transferPathwayView{},
			}
			for _, pw := range prog.Pathways {
				view.Pathways = append(view.Pathways, transferPathwayView{
					Label:                pw.Label,
					ProgramMapID:         pw.ProgramMapID,
					DefaultPathway:       pw.DefaultPathway,
					TransferContinuation: pw.hasNext() || mentionsTransfer(pw.Label),
				})
			}

			// Career outlook: join the program's SOC codes to salary/job-growth.
			if len(prog.CareerSOC) > 0 && !local {
				c, cerr := flags.newClient()
				if cerr != nil {
					return cerr
				}
				raw, _, ferr := c.Post(ctx, "/standard-occupations/batch", map[string]any{"standardOccupationalCodes": prog.CareerSOC})
				if ferr != nil {
					if pmIsRateLimited(ferr) {
						return rateLimitErr(ferr)
					}
					view.Note = "transfer pathways shown; career outlook unavailable: " + ferr.Error()
				} else {
					var batch pmCareersBatch
					if json.Unmarshal(raw, &batch) == nil {
						view.CareerOutlook = &careerOutlookView{
							AvgSalaryK:   batch.BatchAverageSalaryK,
							LowSalaryK:   batch.BatchLowSalaryK,
							HighSalaryK:  batch.BatchHighSalaryK,
							JobGrowthPct: batch.BatchJobGrowthPct,
							Careers:      batch.Careers,
						}
						source = "live"
						view.Source = source
					} else {
						view.Note = "transfer pathways shown; career outlook unavailable: could not parse career data"
					}
				}
			}
			if len(view.Pathways) == 0 && view.CareerOutlook == nil {
				view.Note = "no transfer pathways or career data found for this program"
			}

			if flags.asJSON || (!isTerminal(cmd.OutOrStdout()) && !flags.csv && !flags.quiet && !flags.plain) {
				return printJSONFiltered(cmd.OutOrStdout(), view, flags)
			}
			renderTransferOptions(cmd, view)
			return nil
		},
	}
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	cmd.Flags().StringVar(&siteContent, "site-content", "", "Site-content id (for live fetch when the program is not mirrored)")
	return cmd
}

func isTransferDesignated(p pmProgram) bool {
	if mentionsTransfer(p.AwardTitle) {
		return true
	}
	if p.LinkedPathway {
		return true
	}
	for _, pw := range p.Pathways {
		if mentionsTransfer(pw.Label) || pw.hasNext() {
			return true
		}
	}
	return false
}

func mentionsTransfer(s string) bool {
	l := strings.ToLower(s)
	return strings.Contains(l, "transfer") || strings.Contains(l, "csu") || strings.Contains(l, "uc ") ||
		strings.Contains(l, "cal-getc") || strings.Contains(l, "igetc") || strings.Contains(l, "university")
}

func salaryStr(k *float64) string {
	if k == nil {
		return "n/a"
	}
	return fmt.Sprintf("$%sk", trimFloat(*k))
}

func growthStr(p *float64) string {
	if p == nil {
		return "n/a"
	}
	return fmt.Sprintf("%s%%", trimFloat(*p))
}

func renderTransferOptions(cmd *cobra.Command, view transferOptionsView) {
	w := cmd.OutOrStdout()
	desig := ""
	if view.TransferDesignated {
		desig = " (transfer-designated)"
	}
	fmt.Fprintf(w, "%s — %s%s\n\n", bold(view.Title), view.Award, desig)
	fmt.Fprintf(w, "%s (%d):\n", bold("Transfer pathways"), len(view.Pathways))
	for _, p := range view.Pathways {
		marker := ""
		if p.TransferContinuation {
			marker = " ->"
		}
		fmt.Fprintf(w, "  %s%s\n", p.Label, marker)
	}
	if view.CareerOutlook != nil {
		co := view.CareerOutlook
		fmt.Fprintf(w, "\n%s\n", bold("Career outlook"))
		fmt.Fprintf(w, "  salary: %s – %s (avg %s)   job growth: %s\n",
			salaryStr(co.LowSalaryK), salaryStr(co.HighSalaryK), salaryStr(co.AvgSalaryK), growthStr(co.JobGrowthPct))
		for _, c := range co.Careers {
			fmt.Fprintf(w, "    %-40s avg %s, growth %s\n", c.Title, salaryStr(c.AverageSalaryK), growthStr(c.JobGrowthPct))
		}
	}
	if view.Note != "" {
		fmt.Fprintf(w, "\nnote: %s\n", view.Note)
	}
}
