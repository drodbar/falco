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
