// Package usage shows how a Terraform module repository wires markparsr up as
// its required check: one Go test, run by CI on every pull request.
package usage

import (
	"testing"

	"github.com/codectl/markparsr"
)

func TestReadme(t *testing.T) {
	v, err := markparsr.New(
		markparsr.WithModule("../module"),
		markparsr.WithSections("Goals", "Testing", "Notes"),
		markparsr.WithFiles("GOALS.md", "TESTING.md"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := v.Validate(t.Context()); err != nil {
		t.Fatal(err)
	}
}
