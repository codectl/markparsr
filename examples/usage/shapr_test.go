// Package usage shows how a Terraform module repository wires shapr up as
// its required check: one Go test, run by CI on every pull request.
package usage

import (
	"testing"

	"github.com/codectl/shapr"
)

func TestModule(t *testing.T) {
	v, err := shapr.New(
		shapr.WithModule("../module"),
		shapr.WithSections("Goals", "Testing", "Notes"),
		shapr.WithFiles("GOALS.md", "TESTING.md"),
		shapr.WithSchema(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := v.Validate(t.Context()); err != nil {
		t.Fatal(err)
	}
}
