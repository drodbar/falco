package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ysugimoto/falco/tester/shared"
	"github.com/ysugimoto/falco/token"
)

func TestWriteLCOVToFile(t *testing.T) {
	nodeMap := map[string]token.Token{
		"sub_10_1":  {File: mustAbs(t, "testdata/main.vcl"), Line: 10},
		"stmt_11_3": {File: mustAbs(t, "testdata/main.vcl"), Line: 11},
	}
	factory := &shared.CoverageFactory{
		Subroutines: shared.CoverageFactoryItem{"sub_10_1": 2},
		Statements:  shared.CoverageFactoryItem{"stmt_11_3": 1},
		Branches:    shared.CoverageFactoryItem{},
		NodeMap:     nodeMap,
	}

	out := filepath.Join(t.TempDir(), "lcov.info")
	if err := writeLCOVFile(factory, out); err != nil {
		t.Fatalf("writeLCOVFile: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "SF:") {
		t.Errorf("expected SF: in output, got:\n%s", content)
	}
	if !strings.Contains(content, "end_of_record") {
		t.Errorf("expected end_of_record in output, got:\n%s", content)
	}
}

func mustAbs(t *testing.T, rel string) string {
	t.Helper()
	abs, err := filepath.Abs(rel)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}
