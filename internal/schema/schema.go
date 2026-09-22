// Package schema compares a module's resources and data sources against the
// provider schemas terraform reports, and lists what the module never sets.
package schema

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Check validates the module at root and every modules/<name> directory that
// has a main.tf. exclude lists types to skip: resource types bare, data source
// types prefixed with "data.". Findings are sorted by file, line, and name.
func Check(ctx context.Context, root string, exclude []string) ([]Finding, error) {
	dirs := []string{root}
	entries, err := os.ReadDir(filepath.Join(root, "modules"))
	if err == nil {
		for _, e := range entries {
			sub := filepath.Join(root, "modules", e.Name())
			if _, err := os.Stat(filepath.Join(sub, "main.tf")); e.IsDir() && err == nil {
				dirs = append(dirs, sub)
			}
		}
	}

	var findings []Finding
	for _, dir := range dirs {
		m, err := parseDir(root, dir)
		if err != nil {
			return nil, err
		}
		if len(m.entities) == 0 {
			continue
		}
		schemas, err := loadSchemas(ctx, dir)
		if err != nil {
			return nil, err
		}
		findings = append(findings, compare(m, schemas, exclude)...)
	}

	slices.SortFunc(findings, func(a, b Finding) int {
		return cmp.Or(
			cmp.Compare(a.File, b.File),
			cmp.Compare(a.Line, b.Line),
			cmp.Compare(a.Name, b.Name),
		)
	})
	return findings, nil
}

// Report renders findings grouped by file and resource:
//
//	main.tf: provider schema properties not set
//	line 18: azurerm_virtual_network.this
//	  optional: flow_timeout_in_minutes
//	  required block: delegation
func Report(findings []Finding) string {
	var sb strings.Builder
	file, address := "", ""
	for _, f := range findings {
		if f.File != file {
			file, address = f.File, ""
			fmt.Fprintf(&sb, "%s: provider schema properties not set\n", file)
		}
		if f.Address != address {
			address = f.Address
			fmt.Fprintf(&sb, "line %d: %s\n", f.Line, address)
		}
		label := "optional"
		if f.Required {
			label = "required"
		}
		if f.Block {
			label += " block"
		}
		fmt.Fprintf(&sb, "  %s: %s\n", label, f.Name)
	}
	return strings.TrimRight(sb.String(), "\n")
}
