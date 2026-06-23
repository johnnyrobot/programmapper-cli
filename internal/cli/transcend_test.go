// Copyright 2026 johnnyrobot and contributors. Licensed under Apache-2.0. See LICENSE.
//
// transcend_test.go — hand-authored table-driven tests for the pure-logic
// helpers behind the ProgramMapper transcendence commands (plan, compare,
// diff-years, course-programs, bottlenecks, transfer-options). Fixtures mirror
// the real L.A. Mission response shapes.

package cli

import "testing"

func sampleOpportunity(typ, code, name, courseID string, term int, minU, maxU float64) pmOpportunity {
	op := pmOpportunity{Type: typ, CourseCode: code, CourseName: name, MappedCourseID: courseID, MinUnits: minU, MaxUnits: maxU}
	op.Term.TermNumber = term
	op.Term.Year = 2025
	op.Term.CustomLabel = "Fall"
	return op
}

func element(name, reqType string, op pmOpportunity) pmPathwayElement {
	el := pmPathwayElement{Name: name}
	el.Requirement.RequirementType = reqType
	el.RecommendedOpportunity = op
	return el
}

// nursingMap: 2 terms, two real courses plus a milestone (no units).
func nursingMap() pmMap {
	return pmMap{
		ProgramMapID:      "map-nursing",
		SiteContentID:     "scid",
		TermsToCompletion: 2,
		PMProgramID:       "prog-nursing",
		PMProgramTitle:    "Vocational Nursing",
		PathwayElements: []pmPathwayElement{
			element("NURSING 090", "MAJOR_CORE", sampleOpportunity("COURSE", "NURSING 090", "Skills Lab", "c1", 1, 1, 1)),
			element("BIOLOGY 003", "MAJOR_CORE", sampleOpportunity("COURSE", "BIOLOGY 003", "Anatomy", "c2", 1, 4, 4)),
			element("Apply to program", "MILESTONE", sampleOpportunity("MILESTONE", "", "Apply to program", "", 2, 0, 0)),
			element("MATH 125", "GENERAL_ED", sampleOpportunity("COURSE", "MATH 125", "Algebra", "c3", 2, 3, 3)),
		},
	}
}

// bioMap shares MATH 125 with nursingMap; differs elsewhere.
func bioMap() pmMap {
	return pmMap{
		ProgramMapID:   "map-bio",
		PMProgramID:    "prog-bio",
		PMProgramTitle: "Biology",
		PathwayElements: []pmPathwayElement{
			element("MATH 125", "GENERAL_ED", sampleOpportunity("COURSE", "MATH 125", "Algebra", "c3", 1, 3, 3)),
			element("CHEM 101", "MAJOR_CORE", sampleOpportunity("COURSE", "CHEM 101", "Chemistry", "c9", 1, 5, 5)),
		},
	}
}

func TestPmMapCourses(t *testing.T) {
	got := pmMapCourses(nursingMap())
	if len(got) != 3 {
		t.Fatalf("pmMapCourses: want 3 COURSE entries (milestone excluded), got %d", len(got))
	}
	if _, ok := got["NURSING 090"]; !ok {
		t.Errorf("pmMapCourses: missing NURSING 090")
	}
	if _, ok := got[""]; ok {
		t.Errorf("pmMapCourses: included the empty-code milestone")
	}
}

func TestBuildPlan(t *testing.T) {
	prog := pmProgram{ProgramID: "prog-nursing", Title: "Vocational Nursing", AwardTitle: "Certificate of Achievement"}
	view := buildPlan(prog, nursingMap())
	if view.TotalMinUnits != 8 || view.TotalMaxUnits != 8 {
		t.Errorf("buildPlan total units = %v/%v, want 8/8", view.TotalMinUnits, view.TotalMaxUnits)
	}
	if len(view.Terms) != 2 {
		t.Fatalf("buildPlan terms = %d, want 2", len(view.Terms))
	}
	if view.Terms[0].TermNumber != 1 || view.Terms[1].TermNumber != 2 {
		t.Errorf("buildPlan terms not sorted: %d then %d", view.Terms[0].TermNumber, view.Terms[1].TermNumber)
	}
	if view.Terms[0].MinUnits != 5 {
		t.Errorf("buildPlan term 1 units = %v, want 5 (1+4)", view.Terms[0].MinUnits)
	}
}

func TestBuildCompare(t *testing.T) {
	progA := pmProgram{ProgramID: "prog-nursing", Title: "Vocational Nursing"}
	progB := pmProgram{ProgramID: "prog-bio", Title: "Biology"}
	view := buildCompare(progA, nursingMap(), progB, bioMap())
	if view.SharedCount != 1 || view.Shared[0].Code != "MATH 125" {
		t.Errorf("buildCompare shared = %d %v, want 1 MATH 125", view.SharedCount, view.Shared)
	}
	if len(view.OnlyA) != 2 {
		t.Errorf("buildCompare onlyA = %d, want 2", len(view.OnlyA))
	}
	if len(view.OnlyB) != 1 || view.OnlyB[0].Code != "CHEM 101" {
		t.Errorf("buildCompare onlyB = %v, want [CHEM 101]", view.OnlyB)
	}
}

func TestBuildDiffYears(t *testing.T) {
	prog := pmProgram{ProgramID: "prog-bio", Title: "Biology"}
	// prior = nursingMap (has NURSING 090, BIOLOGY 003, MATH 125), current = bioMap (MATH 125, CHEM 101)
	view := buildDiffYears(prog, bioMap(), nursingMap(), 2022)
	if view.PriorYear != 2022 {
		t.Errorf("prior year = %d, want 2022", view.PriorYear)
	}
	// added (in current, not prior): CHEM 101
	if len(view.Added) != 1 || view.Added[0].Code != "CHEM 101" {
		t.Errorf("added = %v, want [CHEM 101]", view.Added)
	}
	// removed (in prior, not current): NURSING 090, BIOLOGY 003
	if len(view.Removed) != 2 {
		t.Errorf("removed = %v, want 2", view.Removed)
	}
}

func TestIsTransferDesignated(t *testing.T) {
	cases := []struct {
		name string
		prog pmProgram
		want bool
	}{
		{"award says transfer", pmProgram{AwardTitle: "Associate in Science for Transfer"}, true},
		{"linked pathway flag", pmProgram{AwardTitle: "Associate in Arts", LinkedPathway: true}, true},
		{"csu pathway label", pmProgram{Pathways: []pmPathway{{Label: "to CSU - Cal-GETC"}}}, true},
		{"plain certificate", pmProgram{AwardTitle: "Certificate of Achievement"}, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := isTransferDesignated(tc.prog); got != tc.want {
				t.Errorf("isTransferDesignated(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

func TestPmUnitsLabel(t *testing.T) {
	cases := []struct {
		min, max float64
		want     string
	}{
		{1, 1, "1"},
		{1, 4, "1-4"},
		{3.5, 3.5, "3.5"},
		{0, 0, "0"},
	}
	for _, tc := range cases {
		if got := pmUnitsLabel(tc.min, tc.max); got != tc.want {
			t.Errorf("pmUnitsLabel(%v,%v) = %q, want %q", tc.min, tc.max, got, tc.want)
		}
	}
}

func TestDefaultMapID(t *testing.T) {
	p := pmProgram{Pathways: []pmPathway{
		{ProgramMapID: "m1", DefaultPathway: false},
		{ProgramMapID: "m2", DefaultPathway: true},
	}}
	if got := p.defaultMapID(); got != "m2" {
		t.Errorf("defaultMapID = %q, want m2 (default pathway)", got)
	}
	p2 := pmProgram{Pathways: []pmPathway{{ProgramMapID: "m1"}}}
	if got := p2.defaultMapID(); got != "m1" {
		t.Errorf("defaultMapID fallback = %q, want m1", got)
	}
	if got := (pmProgram{}).defaultMapID(); got != "" {
		t.Errorf("defaultMapID empty = %q, want empty", got)
	}
}
