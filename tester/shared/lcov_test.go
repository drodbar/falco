package shared_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ysugimoto/falco/tester/shared"
	"github.com/ysugimoto/falco/token"
)

func newFactory(
	subs, stmts, branches map[string]uint64,
	nodeMap map[string]token.Token,
) *shared.CoverageFactory {
	return &shared.CoverageFactory{
		Subroutines: shared.CoverageFactoryItem(subs),
		Statements:  shared.CoverageFactoryItem(stmts),
		Branches:    shared.CoverageFactoryItem(branches),
		NodeMap:     nodeMap,
	}
}

func TestWriteLCOV_SingleFile_EmptyData(t *testing.T) {
	f := newFactory(
		map[string]uint64{},
		map[string]uint64{},
		map[string]uint64{},
		map[string]token.Token{},
	)
	var buf bytes.Buffer
	if err := f.WriteLCOV(&buf, "/project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output for empty factory, got: %q", buf.String())
	}
}

func TestWriteLCOV_SingleFile_StatementCoverage(t *testing.T) {
	nodeMap := map[string]token.Token{
		"sub_10_1":  {File: "/project/main.vcl", Line: 10},
		"stmt_11_3": {File: "/project/main.vcl", Line: 11},
		"stmt_13_3": {File: "/project/main.vcl", Line: 13},
	}
	f := newFactory(
		map[string]uint64{"sub_10_1": 3},
		map[string]uint64{"stmt_11_3": 2, "stmt_13_3": 0},
		map[string]uint64{},
		nodeMap,
	)

	var buf bytes.Buffer
	if err := f.WriteLCOV(&buf, "/project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()

	mustContain(t, out, "SF:main.vcl")
	mustContain(t, out, "FN:10,sub_10_1")
	mustContain(t, out, "FNDA:3,sub_10_1")
	mustContain(t, out, "FNF:1")
	mustContain(t, out, "FNH:1")
	mustContain(t, out, "DA:10,3")
	mustContain(t, out, "DA:11,2")
	mustContain(t, out, "DA:13,0")
	mustContain(t, out, "LF:3")
	mustContain(t, out, "LH:2")
	mustContain(t, out, "end_of_record")
}

func TestWriteLCOV_BranchCoverage(t *testing.T) {
	nodeMap := map[string]token.Token{
		"sub_10_1":      {File: "/project/main.vcl", Line: 10},
		"stmt_15_5":     {File: "/project/main.vcl", Line: 15},
		"branch_15_5_1": {File: "/project/main.vcl", Line: 15},
		"branch_15_5_2": {File: "/project/main.vcl", Line: 15},
	}
	f := newFactory(
		map[string]uint64{"sub_10_1": 1},
		map[string]uint64{"stmt_15_5": 1},
		map[string]uint64{"branch_15_5_1": 3, "branch_15_5_2": 0},
		nodeMap,
	)

	var buf bytes.Buffer
	if err := f.WriteLCOV(&buf, "/project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()

	mustContain(t, out, "BRDA:15,0,0,3")
	mustContain(t, out, "BRDA:15,0,1,0")
	mustContain(t, out, "BRF:2")
	mustContain(t, out, "BRH:1")
}

func TestWriteLCOV_TernaryBranches(t *testing.T) {
	nodeMap := map[string]token.Token{
		"sub_10_1":          {File: "/project/main.vcl", Line: 10},
		"branch_20_3_true":  {File: "/project/main.vcl", Line: 20},
		"branch_20_3_false": {File: "/project/main.vcl", Line: 20},
	}
	f := newFactory(
		map[string]uint64{"sub_10_1": 1},
		map[string]uint64{},
		map[string]uint64{"branch_20_3_true": 2, "branch_20_3_false": 0},
		nodeMap,
	)

	var buf bytes.Buffer
	if err := f.WriteLCOV(&buf, "/project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()

	mustContain(t, out, "BRDA:20,0,")
	mustContain(t, out, "BRF:2")
	mustContain(t, out, "BRH:1")
}

func TestWriteLCOV_MultipleFiles(t *testing.T) {
	nodeMap := map[string]token.Token{
		"sub_5_1": {File: "/project/main.vcl", Line: 5},
		"sub_3_1": {File: "/project/helpers.vcl", Line: 3},
	}
	f := newFactory(
		map[string]uint64{"sub_5_1": 2, "sub_3_1": 0},
		map[string]uint64{},
		map[string]uint64{},
		nodeMap,
	)

	var buf bytes.Buffer
	if err := f.WriteLCOV(&buf, "/project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()

	mustContain(t, out, "SF:main.vcl")
	mustContain(t, out, "SF:helpers.vcl")
	if count := strings.Count(out, "end_of_record"); count != 2 {
		t.Errorf("expected 2 end_of_record, got %d\n%s", count, out)
	}
}

func TestWriteLCOV_SameLineMergesDA(t *testing.T) {
	nodeMap := map[string]token.Token{
		"sub_10_1":  {File: "/project/main.vcl", Line: 10},
		"stmt_10_5": {File: "/project/main.vcl", Line: 10},
	}
	f := newFactory(
		map[string]uint64{"sub_10_1": 5},
		map[string]uint64{"stmt_10_5": 5},
		map[string]uint64{},
		nodeMap,
	)

	var buf bytes.Buffer
	if err := f.WriteLCOV(&buf, "/project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()

	daCount := strings.Count(out, "DA:10,")
	if daCount != 1 {
		t.Errorf("expected DA:10 to appear once, got %d times\n%s", daCount, out)
	}
	mustContain(t, out, "LF:1")
}

func mustContain(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected output to contain %q\nfull output:\n%s", substr, s)
	}
}
