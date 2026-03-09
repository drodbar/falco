package shared_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ysugimoto/falco/token"
)

func TestWriteGenericXML_Empty(t *testing.T) {
	f := newFactory(
		map[string]uint64{},
		map[string]uint64{},
		map[string]uint64{},
		map[string]token.Token{},
	)
	var buf bytes.Buffer
	if err := f.WriteGenericXML(&buf, "/project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output, got: %q", buf.String())
	}
}

func TestWriteGenericXML_LineCoverage(t *testing.T) {
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
	if err := f.WriteGenericXML(&buf, "/project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()

	mustContain(t, out, `<coverage version="1">`)
	mustContain(t, out, `<file path="main.vcl">`)
	mustContain(t, out, `lineNumber="10" covered="true"`)
	mustContain(t, out, `lineNumber="11" covered="true"`)
	mustContain(t, out, `lineNumber="13" covered="false"`)
	mustContain(t, out, `</coverage>`)
}

func TestWriteGenericXML_BranchCoverage(t *testing.T) {
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
	if err := f.WriteGenericXML(&buf, "/project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()

	mustContain(t, out, `lineNumber="15"`)
	mustContain(t, out, `branchesToCover="2"`)
	mustContain(t, out, `coveredBranches="1"`)
}

func TestWriteGenericXML_MultipleFiles(t *testing.T) {
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
	if err := f.WriteGenericXML(&buf, "/project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()

	mustContain(t, out, `<file path="main.vcl">`)
	mustContain(t, out, `<file path="helpers.vcl">`)
	if count := strings.Count(out, "<file "); count != 2 {
		t.Errorf("expected 2 file elements, got %d\n%s", count, out)
	}
}
