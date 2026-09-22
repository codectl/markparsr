package shapr

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fixture = "examples/module"

func copyFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	entries, err := os.ReadDir(fixture)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(fixture, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, e.Name()), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestNewRequiresModule(t *testing.T) {
	t.Parallel()
	if _, err := New(); err == nil {
		t.Fatal("expected error without WithModule")
	}
}

func TestValidateFixture(t *testing.T) {
	t.Parallel()
	v, err := New(
		WithModule(fixture),
		WithSections("Goals", "Testing", "Notes"),
		WithFiles("GOALS.md", "TESTING.md"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := v.Validate(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestValidateReportsEveryDrift(t *testing.T) {
	t.Parallel()
	dir := copyFixture(t)
	path := filepath.Join(dir, "README.md")
	readme, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	stale := readme
	for _, r := range [][2]string{
		{"## Providers\n", "## Providerss\n"},
		{"- [azurerm_virtual_network.this](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/virtual_network) (resource)\n", ""},
		{"Description: default azure region to be used.", "Description: stale text"},
	} {
		next := bytes.Replace(stale, []byte(r[0]), []byte(r[1]), 1)
		if bytes.Equal(next, stale) {
			t.Fatalf("fixture no longer contains %q", r[0])
		}
		stale = next
	}
	if err := os.WriteFile(path, stale, 0o644); err != nil {
		t.Fatal(err)
	}

	v, err := New(WithModule(dir))
	if err != nil {
		t.Fatal(err)
	}
	err = v.Validate(t.Context())
	if err == nil {
		t.Fatal("expected drift error")
	}
	for _, want := range []string{
		"shapr generate",
		"line 14: differs:\n  README:  ## Providerss\n  sources: ## Providers",
		"line 31: missing from README, generated from Terraform sources:\n  + - [azurerm_virtual_network.this]",
		"differs:\n  README:  Description: stale text\n  sources: Description: default azure region to be used.",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error lacks %q:\n%s", want, err)
		}
	}
}

func TestValidateReportsMissingSectionAndFile(t *testing.T) {
	t.Parallel()
	v, err := New(WithModule(fixture), WithSections("Nope"), WithFiles("NOPE.md"))
	if err != nil {
		t.Fatal(err)
	}
	err = v.Validate(t.Context())
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"section missing: ## Nope", "file missing: NOPE.md"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q lacks %q", err, want)
		}
	}
}

func TestGenerate(t *testing.T) {
	t.Parallel()
	committed, err := os.ReadFile(filepath.Join(fixture, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	stale := bytes.Replace(committed, []byte("Description: default azure region to be used."), []byte("Description: stale"), 1)

	tests := []struct {
		name   string
		readme []byte // nil removes the README before Generate
		want   []byte
	}{
		{"up to date is unchanged", committed, committed},
		{"stale is restored", stale, committed},
		{"absent is created as a bare block", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := copyFixture(t)
			path := filepath.Join(dir, "README.md")
			if tt.readme == nil {
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(path, tt.readme, 0o644); err != nil {
				t.Fatal(err)
			}

			if err := Generate(dir); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if tt.want == nil {
				if !bytes.HasPrefix(got, []byte("<!-- BEGIN_TF_DOCS -->\n## Requirements")) || !bytes.HasSuffix(got, []byte("<!-- END_TF_DOCS -->")) {
					t.Fatalf("fresh README is not a bare block:\n%s", got)
				}
				return
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatal("README differs from committed fixture after Generate")
			}
		})
	}
}
