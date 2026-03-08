# LCOV Coverage Report Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Generate an LCOV-format coverage report file (`lcov.info`) from Falco's existing coverage data, so it can be fed to SonarQube and other tools.

**Architecture:** Add a `WriteLCOV(w io.Writer) error` method to `CoverageFactory` in `tester/shared/lcov.go`, serialising the existing statement/branch/subroutine coverage maps into LCOV format. Wire it up in `cmd/falco/main.go` using the already-declared but unused `CoverageOut` field in `TestConfig`.

**Tech Stack:** Go 1.21+, standard library only (`fmt`, `io`, `os`, `path/filepath`, `sort`, `strconv`, `strings`).

---

## Background: Key Data Structures

Before touching code, read these files:
- `tester/shared/coverage.go` — `CoverageFactory`, `CoverageFactoryItem`, `token.Token`
- `token/token.go` — `Token{Type, Literal, Line, Position, File}`
- `cmd/falco/table.go` — `transformFileMap()` shows how NodeMap is iterated and `.vcl` paths made relative
- `cmd/falco/main.go` lines 414–531 — `runTest()` where coverage output is triggered
- `config/config.go` lines 64–84 — `TestConfig.CoverageOut` (declared, never used)
- `interpreter/coverage.go` — ID format: `sub_<line>_<pos>`, `stmt_<line>_<pos>`, `branch_<line>_<pos>_<suffix>`

## LCOV Format Reference

```
TN:<test name (may be empty)>
SF:<source file path>
FN:<line>,<function name>
FNDA:<count>,<function name>
FNF:<functions found>
FNH:<functions hit>
BRDA:<line>,<block>,<branch>,<count>
BRF:<branches found>
BRH:<branches hit>
DA:<line>,<count>
LF:<lines found>
LH:<lines hit>
end_of_record
```

## ID Format Reference (from `interpreter/coverage.go`)

| Type | Format | Example |
|------|--------|---------|
| Subroutine | `sub_<line>_<pos>` | `sub_10_1` |
| Statement | `stmt_<line>_<pos>` | `stmt_11_3` |
| Branch | `branch_<line>_<pos>_<suffix>` | `branch_15_5_1`, `branch_15_5_true` |

Branch suffixes: `1`, `2`, `3`… for if/else-if/else chains; `true`/`false` for ternary `if()` expressions.

---

## Task 1: Fix typo in `cmd/falco/table.go` (bug)

**Files:**
- Modify: `cmd/falco/table.go:163`

**Step 1: Verify the bug**

Read line 163 of `cmd/falco/table.go`. You will see:
```go
case strings.HasPrefix(id, "brancn"):
```
This typo means branch coverage is never grouped by file in the console table.

**Step 2: Fix the typo**

Change `"brancn"` to `"branch"`:
```go
case strings.HasPrefix(id, "branch"):
```

**Step 3: Build to verify no compilation error**

```bash
go build ./cmd/falco
```
Expected: no output (success).

**Step 4: Commit**

```bash
git add cmd/falco/table.go
git commit -m "fix: correct branch prefix typo in transformFileMap"
```

---

## Task 2: Write failing tests for LCOV generation

**Files:**
- Create: `tester/shared/lcov_test.go`

**Step 1: Write the test file**

```go
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
		"sub_10_1":       {File: "/project/main.vcl", Line: 10},
		"stmt_15_5":      {File: "/project/main.vcl", Line: 15},
		"branch_15_5_1":  {File: "/project/main.vcl", Line: 15},
		"branch_15_5_2":  {File: "/project/main.vcl", Line: 15},
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
		"sub_10_1":           {File: "/project/main.vcl", Line: 10},
		"branch_20_3_true":   {File: "/project/main.vcl", Line: 20},
		"branch_20_3_false":  {File: "/project/main.vcl", Line: 20},
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
		"sub_5_1":  {File: "/project/main.vcl", Line: 5},
		"sub_3_1":  {File: "/project/helpers.vcl", Line: 3},
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
```

**Step 2: Run the tests to confirm they fail**

```bash
cd /home/drodriguez/workspace/falco && go test ./tester/shared/... -run TestWriteLCOV -v
```
Expected: **FAIL** — `f.WriteLCOV undefined`.

**Step 3: Commit the failing tests**

```bash
git add tester/shared/lcov_test.go
git commit -m "test(coverage): add failing tests for LCOV report generation"
```

---

## Task 3: Implement `WriteLCOV` in `tester/shared/lcov.go`

**Files:**
- Create: `tester/shared/lcov.go`

**Step 1: Create the implementation**

```go
package shared

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/ysugimoto/falco/token"
)

// WriteLCOV writes coverage in LCOV trace format to w.
// Pass empty baseDir to use os.Getwd().
func (c *CoverageFactory) WriteLCOV(w io.Writer, baseDir string) error {
	if baseDir == "" {
		var err error
		baseDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("lcov: getwd: %w", err)
		}
	}

	groups, err := c.groupByFile(baseDir)
	if err != nil {
		return err
	}
	if len(groups) == 0 {
		return nil
	}

	files := make([]string, 0, len(groups))
	for f := range groups {
		files = append(files, f)
	}
	sort.Strings(files)

	for _, file := range files {
		if err := groups[file].writeRecord(w, file); err != nil {
			return err
		}
	}
	return nil
}

type lcovFileGroup struct {
	subroutines map[string]uint64
	statements  map[string]uint64
	branches    map[string]uint64
	nodeMap     map[string]token.Token
}

func newLCOVFileGroup() *lcovFileGroup {
	return &lcovFileGroup{
		subroutines: make(map[string]uint64),
		statements:  make(map[string]uint64),
		branches:    make(map[string]uint64),
		nodeMap:     make(map[string]token.Token),
	}
}

func (c *CoverageFactory) groupByFile(baseDir string) (map[string]*lcovFileGroup, error) {
	result := make(map[string]*lcovFileGroup)

	addID := func(file, id string, tok token.Token) *lcovFileGroup {
		g, ok := result[file]
		if !ok {
			g = newLCOVFileGroup()
			result[file] = g
		}
		g.nodeMap[id] = tok
		return g
	}

	for id, tok := range c.NodeMap {
		file, err := lcovFilePath(tok, baseDir)
		if err != nil {
			return nil, err
		}
		g := addID(file, id, tok)

		switch {
		case strings.HasPrefix(id, "sub_"):
			g.subroutines[id] = c.Subroutines[id]
		case strings.HasPrefix(id, "stmt_"):
			g.statements[id] = c.Statements[id]
		case strings.HasPrefix(id, "branch_"):
			g.branches[id] = c.Branches[id]
		}
	}
	return result, nil
}

func lcovFilePath(tok token.Token, baseDir string) (string, error) {
	if strings.EqualFold(filepath.Ext(tok.File), ".vcl") {
		rel, err := filepath.Rel(baseDir, tok.File)
		if err != nil {
			return "", fmt.Errorf("lcov: rel path for %q: %w", tok.File, err)
		}
		return rel, nil
	}
	return tok.File, nil
}

func (g *lcovFileGroup) writeRecord(w io.Writer, file string) error {
	fmt.Fprintln(w, "TN:")
	fmt.Fprintf(w, "SF:%s\n", file)

	subIDs := sortedKeys(g.subroutines)
	for _, id := range subIDs {
		fmt.Fprintf(w, "FN:%d,%s\n", g.nodeMap[id].Line, id)
	}
	for _, id := range subIDs {
		fmt.Fprintf(w, "FNDA:%d,%s\n", g.subroutines[id], id)
	}
	fmt.Fprintf(w, "FNF:%d\n", len(g.subroutines))
	fmt.Fprintf(w, "FNH:%d\n", countHit(g.subroutines))

	// Group branch IDs by (line, pos) to assign stable block numbers.
	type blockKey struct{ line, pos int }
	blockOrder := []blockKey{}
	blockMap := map[blockKey][]string{}

	branchIDs := sortedKeys(g.branches)
	for _, id := range branchIDs {
		line, pos := parseLCOVLinePos(id)
		key := blockKey{line, pos}
		if _, exists := blockMap[key]; !exists {
			blockOrder = append(blockOrder, key)
		}
		blockMap[key] = append(blockMap[key], id)
	}
	sort.Slice(blockOrder, func(i, j int) bool {
		if blockOrder[i].line != blockOrder[j].line {
			return blockOrder[i].line < blockOrder[j].line
		}
		return blockOrder[i].pos < blockOrder[j].pos
	})

	for blockIdx, key := range blockOrder {
		for branchIdx, id := range blockMap[key] {
			count := g.branches[id]
			fmt.Fprintf(w, "BRDA:%d,%d,%d,%d\n", key.line, blockIdx, branchIdx, count)
		}
	}
	fmt.Fprintf(w, "BRF:%d\n", len(g.branches))
	fmt.Fprintf(w, "BRH:%d\n", countHit(g.branches))

	// Merge statements and subroutine lines; same line takes the max count.
	lineMap := make(map[int]uint64)
	for id, count := range g.statements {
		line := g.nodeMap[id].Line
		if existing, ok := lineMap[line]; !ok || count > existing {
			lineMap[line] = count
		}
	}
	for id, count := range g.subroutines {
		line := g.nodeMap[id].Line
		if existing, ok := lineMap[line]; !ok || count > existing {
			lineMap[line] = count
		}
	}

	lineNums := make([]int, 0, len(lineMap))
	for ln := range lineMap {
		lineNums = append(lineNums, ln)
	}
	sort.Ints(lineNums)

	linesHit := 0
	for _, ln := range lineNums {
		count := lineMap[ln]
		fmt.Fprintf(w, "DA:%d,%d\n", ln, count)
		if count > 0 {
			linesHit++
		}
	}
	fmt.Fprintf(w, "LF:%d\n", len(lineMap))
	fmt.Fprintf(w, "LH:%d\n", linesHit)

	fmt.Fprintln(w, "end_of_record")
	return nil
}

func sortedKeys(m map[string]uint64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func countHit(m map[string]uint64) int {
	n := 0
	for _, v := range m {
		if v > 0 {
			n++
		}
	}
	return n
}

func parseLCOVLinePos(id string) (line, pos int) {
	parts := strings.Split(id, "_")
	if len(parts) < 3 {
		return 0, 0
	}
	line, _ = strconv.Atoi(parts[1])
	pos, _ = strconv.Atoi(parts[2])
	return line, pos
}
```

**Step 2: Run the tests**

```bash
cd /home/drodriguez/workspace/falco && go test ./tester/shared/... -run TestWriteLCOV -v
```
Expected: all `TestWriteLCOV_*` tests **PASS**.

**Step 3: Run full test suite to check for regressions**

```bash
make test
```
Expected: all tests pass.

**Step 4: Commit**

```bash
git add tester/shared/lcov.go tester/shared/lcov_test.go
git commit -m "feat(coverage): implement WriteLCOV for LCOV report generation"
```

---

## Task 4: Wire up `--coverage-out` in the CLI

**Files:**
- Modify: `cmd/falco/main.go` (lines 518–525, inside `runTest`)

**Step 1: Write a failing integration test**

Create `cmd/falco/lcov_integration_test.go`:

```go
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
```

**Step 2: Run to confirm it fails**

```bash
cd /home/drodriguez/workspace/falco && go test ./cmd/falco/... -run TestWriteLCOVToFile -v
```
Expected: **FAIL** — `writeLCOVFile undefined`.

**Step 3: Add `writeLCOVFile` helper and wire it in `runTest`**

Create `cmd/falco/lcov.go`:

```go
package main

import (
	"os"

	"github.com/pkg/errors"
	"github.com/ysugimoto/falco/tester/shared"
)

func writeLCOVFile(factory *shared.CoverageFactory, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return errors.WithStack(err)
	}
	defer f.Close()
	return errors.WithStack(factory.WriteLCOV(f, ""))
}
```

Then modify `cmd/falco/main.go` inside `runTest`, after the `printCoverageTable` block (lines 518–525). The existing block is:

```go
	if factory.Coverage != nil {
		writeln(white, "")
		writeln(white, "Coverage Report")
		if err := printCoverageTable(factory.Coverage); err != nil {
			writeln(red, err.Error())
			return ErrExit
		}
	}
```

Replace it with:

```go
	if factory.Coverage != nil {
		writeln(white, "")
		writeln(white, "Coverage Report")
		if err := printCoverageTable(factory.Coverage); err != nil {
			writeln(red, err.Error())
			return ErrExit
		}
		if out := runner.config.Testing.CoverageOut; out != "" {
			if err := writeLCOVFile(factory.Coverage, out); err != nil {
				writeln(red, "Failed to write coverage report: %s", err.Error())
				return ErrExit
			}
			writeln(white, "Coverage report written to %s", out)
		}
	}
```

**Step 4: Run the integration test**

```bash
cd /home/drodriguez/workspace/falco && go test ./cmd/falco/... -run TestWriteLCOVToFile -v
```
Expected: **PASS**.

**Step 5: Build and smoke-test the binary**

```bash
go build ./cmd/falco
```
Expected: compiles cleanly.

**Step 6: Run full test suite**

```bash
make test
```
Expected: all tests pass.

**Step 7: Commit**

```bash
git add cmd/falco/lcov.go cmd/falco/lcov_integration_test.go cmd/falco/main.go
git commit -m "feat(coverage): wire --coverage-out flag to write LCOV report file"
```

---

## Verification

After all tasks are complete, verify end-to-end:

```bash
# Run tests with coverage and LCOV output
./falco test --coverage --coverage-out lcov.info -I . /path/to/main.vcl

# Confirm file exists and has expected structure
head -20 lcov.info
```

Expected `lcov.info` content:
```
TN:
SF:main.vcl
FN:10,sub_10_1
FNDA:3,sub_10_1
FNF:1
FNH:1
...
DA:10,3
...
end_of_record
```

---

## Notes for the Implementer

1. **`CoverageOut` is already declared** in `config/config.go:73` as `cli:"coverage-out"` — it just needs to be used. No config changes required.

2. **The `WriteLCOV` signature takes a `baseDir` parameter** (not `os.Getwd()` inline) so it is testable without depending on the working directory.

3. **`TN:` (test name)** is intentionally left empty — LCOV allows empty test names and SonarQube does not require them.

4. **Branch `BRDA` counts use `0`** (not `-`) for not-taken branches, because all instrumented branches are reachable — `-` in LCOV means "unreachable", which would suppress them in reports.

5. **`DA:` merges subroutine and statement markers** on the same line by taking the maximum count, avoiding duplicate line entries that would confuse LCOV parsers.

6. **Snippet files** (e.g. `snippet::foo`) are passed through as-is to `SF:`, same as the existing table output does. SonarQube will ignore files it cannot map to source, which is acceptable.
