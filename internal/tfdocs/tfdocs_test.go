package tfdocs

import (
	"bytes"
	"errors"
	"os"
	"testing"
)

const fixture = "../../examples/module"

func TestRenderMatchesCommittedBlock(t *testing.T) {
	t.Parallel()
	readme, err := os.ReadFile(fixture + "/README.md")
	if err != nil {
		t.Fatal(err)
	}
	want, err := Block(readme)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Render(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("rendered block differs from committed README block\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestRenderMissingDir(t *testing.T) {
	t.Parallel()
	if _, err := Render(t.TempDir() + "/nope"); err == nil {
		t.Fatal("expected error for missing module dir")
	}
}

func TestBlock(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		readme  string
		want    string
		wantErr error
	}{
		{"present", "# T\n\n<!-- BEGIN_TF_DOCS -->\nbody\n<!-- END_TF_DOCS -->\n\n## After\n", "body", nil},
		{"empty body", "<!-- BEGIN_TF_DOCS -->\n<!-- END_TF_DOCS -->", "", nil},
		{"no markers", "# T\n", "", ErrNoBlock},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Block([]byte(tt.readme))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if string(got) != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}

	for _, readme := range []string{
		"<!-- END_TF_DOCS -->",
		"<!-- BEGIN_TF_DOCS -->",
		"<!-- END_TF_DOCS -->\n<!-- BEGIN_TF_DOCS -->",
	} {
		if _, err := Block([]byte(readme)); err == nil || errors.Is(err, ErrNoBlock) {
			t.Errorf("Block(%q) err = %v, want a marker error", readme, err)
		}
	}
}

func TestInject(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		readme string
		want   string
	}{
		{"empty readme", "", "<!-- BEGIN_TF_DOCS -->\nnew\n<!-- END_TF_DOCS -->"},
		{"no markers appends", "# T\n", "# T\n\n<!-- BEGIN_TF_DOCS -->\nnew\n<!-- END_TF_DOCS -->"},
		{"replaces in place", "# T\n\n<!-- BEGIN_TF_DOCS -->\nold\n<!-- END_TF_DOCS -->\n\n## After\n", "# T\n\n<!-- BEGIN_TF_DOCS -->\nnew\n<!-- END_TF_DOCS -->\n\n## After\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Inject([]byte(tt.readme), []byte("new"))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
			block, err := Block(got)
			if err != nil || string(block) != "new" {
				t.Fatalf("Block(Inject()) = %q, %v; want %q", block, err, "new")
			}
		})
	}

	if _, err := Inject([]byte("<!-- END_TF_DOCS -->"), []byte("x")); err == nil {
		t.Fatal("expected error on malformed markers")
	}
}
