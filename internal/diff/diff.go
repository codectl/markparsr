// Package diff explains, in words, how the README block a module has differs
// from the block its Terraform sources generate.
package diff

import (
	"bytes"
	"fmt"
	"strings"
)

// Report lists every change needed to turn have into want, or returns "" when
// they are equal. Each change says whether lines are missing from the README,
// present in the README but not in the sources, or differ, and shows the lines
// on each side. Line numbers refer to have and start at offset so callers can
// map them to the README the block was cut from.
func Report(have, want []byte, offset int) string {
	if bytes.Equal(have, want) {
		return ""
	}
	ops := edits(bytes.Split(have, []byte("\n")), bytes.Split(want, []byte("\n")))

	var sb strings.Builder
	for i := 0; i < len(ops); {
		if ops[i].kind == keep {
			i++
			continue
		}
		// One change = a run of removes followed by a run of adds.
		var readme, sources [][]byte
		line := ops[i].line + offset
		for ; i < len(ops) && ops[i].kind == remove; i++ {
			readme = append(readme, ops[i].text)
		}
		for ; i < len(ops) && ops[i].kind == add; i++ {
			sources = append(sources, ops[i].text)
		}

		switch {
		case len(readme) == 0:
			fmt.Fprintf(&sb, "line %d: missing from README, generated from Terraform sources:\n", line)
			writeLines(&sb, "  + ", sources)
		case len(sources) == 0:
			fmt.Fprintf(&sb, "line %d: in README, not generated from Terraform sources:\n", line)
			writeLines(&sb, "  - ", readme)
		default:
			fmt.Fprintf(&sb, "line %d: differs:\n", line)
			writeLines(&sb, "  README:  ", readme)
			writeLines(&sb, "  sources: ", sources)
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

func writeLines(sb *strings.Builder, prefix string, lines [][]byte) {
	for _, l := range lines {
		if len(bytes.TrimSpace(l)) == 0 {
			continue
		}
		sb.WriteString(prefix)
		sb.Write(l)
		sb.WriteByte('\n')
	}
}

type kind byte

const (
	keep kind = iota
	remove
	add
)

type edit struct {
	kind kind
	text []byte
	line int // 0-based line in have; for add, the line it precedes
}

// edits computes a minimal edit script via the LCS table, emitting removes
// before adds at each change. Blocks are a few hundred lines, so the
// quadratic table is negligible.
func edits(a, b [][]byte) []edit {
	n, m := len(a), len(b)
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if bytes.Equal(a[i], b[j]) {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}

	var ops []edit
	i, j := 0, 0
	for i < n || j < m {
		switch {
		case i < n && j < m && bytes.Equal(a[i], b[j]):
			ops = append(ops, edit{keep, a[i], i})
			i++
			j++
		case i < n && (j == m || lcs[i+1][j] >= lcs[i][j+1]):
			ops = append(ops, edit{remove, a[i], i})
			i++
		default:
			ops = append(ops, edit{add, b[j], i})
			j++
		}
	}
	return ops
}
