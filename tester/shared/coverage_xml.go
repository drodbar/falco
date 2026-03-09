package shared

import (
	"fmt"
	"io"
	"os"
	"sort"
)

// WriteGenericXML writes coverage in SonarQube Generic Coverage XML format to w.
// Pass empty baseDir to use os.Getwd().
func (c *CoverageFactory) WriteGenericXML(w io.Writer, baseDir string) error {
	if baseDir == "" {
		var err error
		baseDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("generic-xml: getwd: %w", err)
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

	fmt.Fprintln(w, `<?xml version="1.0" ?>`)
	fmt.Fprintln(w, `<coverage version="1">`)
	for _, file := range files {
		if err := groups[file].writeXMLFile(w, file); err != nil {
			return err
		}
	}
	fmt.Fprintln(w, `</coverage>`)
	return nil
}

func (g *lcovFileGroup) writeXMLFile(w io.Writer, file string) error {
	fmt.Fprintf(w, "  <file path=%q>\n", file)

	lineMap := make(map[int]uint64)
	for id, count := range g.statements {
		ln := g.nodeMap[id].Line
		if existing, ok := lineMap[ln]; !ok || count > existing {
			lineMap[ln] = count
		}
	}
	for id, count := range g.subroutines {
		ln := g.nodeMap[id].Line
		if existing, ok := lineMap[ln]; !ok || count > existing {
			lineMap[ln] = count
		}
	}

	type branchStat struct{ total, hit int }
	branchMap := make(map[int]*branchStat)
	for id, count := range g.branches {
		ln := g.nodeMap[id].Line
		if branchMap[ln] == nil {
			branchMap[ln] = &branchStat{}
		}
		branchMap[ln].total++
		if count > 0 {
			branchMap[ln].hit++
		}
	}

	lineNums := make([]int, 0, len(lineMap))
	for ln := range lineMap {
		lineNums = append(lineNums, ln)
	}
	sort.Ints(lineNums)

	for _, ln := range lineNums {
		covered := lineMap[ln] > 0
		if bs, ok := branchMap[ln]; ok {
			fmt.Fprintf(w,
				"    <lineToCover lineNumber=%q covered=%q branchesToCover=%q coveredBranches=%q/>\n",
				fmt.Sprint(ln), fmt.Sprint(covered), fmt.Sprint(bs.total), fmt.Sprint(bs.hit),
			)
		} else {
			fmt.Fprintf(w, "    <lineToCover lineNumber=%q covered=%q/>\n", fmt.Sprint(ln), fmt.Sprint(covered))
		}
	}

	fmt.Fprintln(w, "  </file>")
	return nil
}
