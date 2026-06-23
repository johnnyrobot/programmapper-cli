// Copyright 2026 johnnyrobot and contributors. Licensed under Apache-2.0. See LICENSE.
//
// mirror.go — hand-authored. Walks a college's ProgramMapper catalog graph into
// the local SQLite store so the transcendence commands (plan, compare,
// course-programs, bottlenecks, diff-years, transfer-options) and the framework
// `search` can run offline. ProgramMapper has no flat list endpoints, so this is
// the only way to populate the local mirror.
//
// pp:data-source live

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"programmapper-pp-cli/internal/client"
	"programmapper-pp-cli/internal/cliutil"
)

func newMirrorCmd(flags *rootFlags) *cobra.Command {
	var dbPath string
	var maxPrograms int
	var withMaps bool

	cmd := &cobra.Command{
		Use:   "mirror <college>",
		Short: "Download a college's catalog (clusters, programs, maps, courses) into local SQLite",
		Long: strings.TrimSpace(`
Download a college's full ProgramMapper catalog into the local SQLite store so
offline commands (search, plan, compare, course-programs, bottlenecks) can run
without hitting the API.

<college> is a ProgramMapper vanity URL (https://la-mission.programmapper.ws) or
a college registry id (la_mission). The light pass (default) mirrors interest
clusters and program summaries. Add --maps to also fetch each program's map and
courses (needed by plan, compare, course-programs, and bottlenecks); this makes
many paced requests and is resumable, so re-running continues where it stopped.`),
		Example: strings.Trim(`
  programmapper-pp-cli mirror https://la-mission.programmapper.ws
  programmapper-pp-cli mirror la_mission --maps --max-programs 25
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "false"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would mirror college catalog into local SQLite")
				return nil
			}
			if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("a college (vanity URL or registry id) is required"))
			}
			college := args[0]

			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			// Under live dogfood, keep the matrix fast and avoid re-tripping the
			// WAF: cap deep mirroring to a couple programs.
			if cliutil.IsDogfoodEnv() {
				if withMaps && (maxPrograms == 0 || maxPrograms > 2) {
					maxPrograms = 2
				}
			}

			dbPath = pmDBPath(dbPath)
			db, err := pmOpenStore(dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()

			limiter := cliutil.NewAdaptiveLimiter(4)
			stats := &mirrorStats{}

			scid, _, err := pmResolveSiteContent(ctx, c, college)
			if err != nil {
				if pmIsRateLimited(err) {
					return rateLimitErr(fmt.Errorf("rate-limited resolving college %q: %w", college, err))
				}
				return classifyAPIError(err, flags)
			}

			// Site-content config.
			if raw, err := pmGetThrottled(ctx, c, limiter, fmt.Sprintf("/site-contents/%s", scid)); err == nil {
				if uerr := db.Upsert(rtSiteContents, scid, raw); uerr != nil {
					mirrorWarn(cmd, "storing site-content %s: %v", scid, uerr)
				} else {
					stats.siteContents++
				}
			} else if pmIsRateLimited(err) {
				return rateLimitErr(fmt.Errorf("rate-limited fetching site-content: %w", err))
			} else {
				mirrorWarn(cmd, "site-content config %s: %v", scid, err)
			}

			// Home page -> clusters.
			homeRaw, err := pmGetThrottled(ctx, c, limiter, fmt.Sprintf("/site-contents/%s/home-page-content", scid))
			if err != nil {
				if pmIsRateLimited(err) {
					return rateLimitErr(fmt.Errorf("rate-limited fetching home page: %w", err))
				}
				return classifyAPIError(err, flags)
			}
			var home pmHomePage
			if jerr := json.Unmarshal(homeRaw, &home); jerr != nil {
				return fmt.Errorf("parsing home-page-content: %w", jerr)
			}

			programIDs := make([]string, 0, 256)
			seenProgram := map[string]bool{}
			for _, cluster := range home.ProgramGroups {
				if cluster.MasterRecordID == "" {
					continue
				}
				// Fetch the cluster (program-group) detail to enumerate programs.
				clRaw, err := pmGetThrottled(ctx, c, limiter, fmt.Sprintf("/site-contents/%s/program-groups/%s", scid, cluster.MasterRecordID))
				if err != nil {
					if pmIsRateLimited(err) {
						return rateLimitErr(fmt.Errorf("rate-limited fetching cluster %s: %w", cluster.MasterRecordID, err))
					}
					mirrorWarn(cmd, "cluster %s: %v", cluster.MasterRecordID, err)
					continue
				}
				_ = db.Upsert(rtClusters, cluster.MasterRecordID, clRaw)
				stats.clusters++
				var cl pmCluster
				if json.Unmarshal(clRaw, &cl) != nil {
					continue
				}
				for _, ps := range cl.Programs {
					if ps.MasterRecordID == "" || seenProgram[ps.MasterRecordID] {
						continue
					}
					seenProgram[ps.MasterRecordID] = true
					programIDs = append(programIDs, ps.MasterRecordID)
					// Store the summary (enriched later by detail under --maps).
					sumRaw, _ := json.Marshal(map[string]any{
						"programId":         ps.MasterRecordID,
						"title":             ps.Title,
						"awardTitle":        ps.AwardShortTitle,
						"awardCategory":     ps.AwardCategory,
						"transferAvailable": ps.TransferAvailable,
						"siteContentId":     scid,
						"clusterId":         cluster.MasterRecordID,
						"clusterTitle":      strings.TrimSpace(cluster.Title),
					})
					if existing, _, _ := pmGetProgram(db, ps.MasterRecordID); existing.ProgramID == "" {
						_ = db.Upsert(rtPrograms, ps.MasterRecordID, sumRaw)
					}
					stats.programs++
				}
			}

			recordMirrorSyncState(db, stats)
			mirrorProgress(cmd, "mirrored %d clusters, %d programs (rate %.1f req/s)", stats.clusters, stats.programs, limiter.Rate())

			if !withMaps {
				return finishMirror(cmd, flags, dbPath, stats)
			}

			// Deep pass: program detail + default map + courses.
			built := 0
			for _, pid := range programIDs {
				if maxPrograms > 0 && built >= maxPrograms {
					stats.capped = true
					break
				}
				// Resumable: skip programs whose default map is already stored.
				if existing, ok, _ := pmGetProgram(db, pid); ok && existing.defaultMapID() != "" {
					if _, mok, _ := pmGetMap(db, existing.defaultMapID()); mok {
						continue
					}
				}
				prog, err := pmFetchProgramThrottled(ctx, c, limiter, scid, pid)
				if err != nil {
					if pmIsRateLimited(err) {
						mirrorWarn(cmd, "rate-limited at program %s; stopping deep pass with %d maps (re-run mirror to resume)", pid, stats.maps)
						stats.rateLimited = true
						break
					}
					mirrorWarn(cmd, "program %s: %v", pid, err)
					continue
				}
				progRaw, _ := json.Marshal(prog)
				_ = db.Upsert(rtPrograms, pid, progRaw)
				built++

				mapID := prog.defaultMapID()
				if mapID == "" {
					continue
				}
				m, err := pmFetchMapThrottled(ctx, c, limiter, scid, mapID)
				if err != nil {
					if pmIsRateLimited(err) {
						mirrorWarn(cmd, "rate-limited fetching map for %s; stopping deep pass (re-run mirror to resume)", pid)
						stats.rateLimited = true
						break
					}
					mirrorWarn(cmd, "map %s: %v", mapID, err)
					continue
				}
				m.PMProgramID = prog.ProgramID
				if m.PMProgramID == "" {
					m.PMProgramID = pid
				}
				m.PMProgramTitle = prog.Title
				m.PMAward = prog.AwardTitle
				mRaw, _ := json.Marshal(m)
				if uerr := db.Upsert(rtProgramMaps, mapID, mRaw); uerr != nil {
					mirrorWarn(cmd, "storing map %s: %v", mapID, uerr)
					continue
				}
				stats.maps++

				// Derive course rows from the map's opportunities (lightweight;
				// `courses get` fetches full detail live when needed).
				for _, el := range m.PathwayElements {
					op := el.RecommendedOpportunity
					if op.Type != "COURSE" || op.MappedCourseID == "" {
						continue
					}
					if _, err := db.Get(rtCourses, op.MappedCourseID); err == nil {
						continue // already stored
					}
					courseRaw, _ := json.Marshal(map[string]any{
						"masterRecordId": op.MappedCourseID,
						"courseCode":     op.CourseCode,
						"title":          op.CourseName,
						"minUnits":       op.MinUnits,
						"maxUnits":       op.MaxUnits,
						"siteContentId":  scid,
					})
					_ = db.Upsert(rtCourses, op.MappedCourseID, courseRaw)
					stats.courses++
				}
				if stats.maps%10 == 0 {
					mirrorProgress(cmd, "deep mirror: %d maps, %d courses (rate %.1f req/s)", stats.maps, stats.courses, limiter.Rate())
				}
			}

			recordMirrorSyncState(db, stats)
			return finishMirror(cmd, flags, dbPath, stats)
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Database path (default: standard data dir)")
	cmd.Flags().BoolVar(&withMaps, "maps", false, "Also fetch each program's map and courses (needed by plan/compare/course-programs/bottlenecks)")
	cmd.Flags().IntVar(&maxPrograms, "max-programs", 0, "With --maps, cap how many programs to deep-mirror (0 = all)")
	return cmd
}

type mirrorStats struct {
	siteContents int
	clusters     int
	programs     int
	maps         int
	courses      int
	capped       bool
	rateLimited  bool
}

func pmGetThrottled(ctx context.Context, c *client.Client, l *cliutil.AdaptiveLimiter, path string) (json.RawMessage, error) {
	l.Wait()
	raw, err := c.Get(ctx, path, nil)
	if err != nil {
		if pmIsRateLimited(err) {
			l.OnRateLimit()
		}
		return nil, err
	}
	l.OnSuccess()
	return raw, nil
}

func pmFetchProgramThrottled(ctx context.Context, c *client.Client, l *cliutil.AdaptiveLimiter, scid, pid string) (pmProgram, error) {
	l.Wait()
	p, err := pmFetchProgram(ctx, c, scid, pid)
	if err != nil {
		if pmIsRateLimited(err) {
			l.OnRateLimit()
		}
		return pmProgram{}, err
	}
	l.OnSuccess()
	return p, nil
}

func pmFetchMapThrottled(ctx context.Context, c *client.Client, l *cliutil.AdaptiveLimiter, scid, mapID string) (pmMap, error) {
	l.Wait()
	m, err := pmFetchMap(ctx, c, scid, mapID)
	if err != nil {
		if pmIsRateLimited(err) {
			l.OnRateLimit()
		}
		return pmMap{}, err
	}
	l.OnSuccess()
	return m, nil
}

func mirrorWarn(cmd *cobra.Command, format string, a ...any) {
	fmt.Fprintf(cmd.ErrOrStderr(), "warning: "+format+"\n", a...)
}

func mirrorProgress(cmd *cobra.Command, format string, a ...any) {
	fmt.Fprintf(cmd.ErrOrStderr(), format+"\n", a...)
}

type mirrorResult struct {
	DBPath       string `json:"db_path"`
	SiteContents int    `json:"site_contents"`
	Clusters     int    `json:"clusters"`
	Programs     int    `json:"programs"`
	Maps         int    `json:"maps"`
	Courses      int    `json:"courses"`
	Capped       bool   `json:"capped"`
	RateLimited  bool   `json:"rate_limited"`
	Note         string `json:"note,omitempty"`
}

func finishMirror(cmd *cobra.Command, flags *rootFlags, dbPath string, s *mirrorStats) error {
	res := mirrorResult{
		DBPath:       dbPath,
		SiteContents: s.siteContents,
		Clusters:     s.clusters,
		Programs:     s.programs,
		Maps:         s.maps,
		Courses:      s.courses,
		Capped:       s.capped,
		RateLimited:  s.rateLimited,
	}
	if s.rateLimited {
		res.Note = "deep pass stopped early on a rate limit; re-run mirror --maps to resume"
	} else if s.capped {
		res.Note = "deep pass capped by --max-programs; raise it to mirror more"
	} else if s.maps == 0 {
		res.Note = "light mirror only; add --maps to enable plan/compare/course-programs/bottlenecks"
	}
	if flags.asJSON || flags.agent {
		return printJSONFiltered(cmd.OutOrStdout(), res, flags)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Mirrored to %s\n  clusters: %d\n  programs: %d\n  maps:     %d\n  courses:  %d\n", dbPath, res.Clusters, res.Programs, res.Maps, res.Courses)
	if res.Note != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "  note: %s\n", res.Note)
	}
	return nil
}
