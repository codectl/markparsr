package check

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "full.tf"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "empty.tf"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Files(dir, "full.tf"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := Files(dir, "full.tf", "empty.tf", "missing.tf")
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"file empty: empty.tf", "file missing: missing.tf"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q lacks %q", err, want)
		}
	}
	if strings.Contains(err.Error(), "full.tf") {
		t.Errorf("error %q wrongly mentions full.tf", err)
	}
}

func TestSections(t *testing.T) {
	t.Parallel()
	readme := []byte("# Title\n\n## Goals\n\ntext\n\n### Testing\n\n## Notes extra\n")

	if err := Sections(readme, "Goals"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := Sections(readme, "Goals", "Testing", "Notes")
	if err == nil {
		t.Fatal("expected error")
	}
	// H3 and a heading with trailing words must not satisfy an H2 requirement.
	for _, want := range []string{"section missing: ## Testing", "section missing: ## Notes"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q lacks %q", err, want)
		}
	}
}

func TestURLs(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	readme := []byte("see " + srv.URL + "/ok and " + srv.URL + "/missing " +
		"and https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/subnet")

	err := URLs(t.Context(), srv.Client(), readme)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "/missing: status 404") {
		t.Errorf("error %q lacks 404 report", err)
	}
	if strings.Contains(err.Error(), "/ok") || strings.Contains(err.Error(), "registry.terraform.io") {
		t.Errorf("error %q reports a URL that should pass or be skipped", err)
	}

	if err := URLs(t.Context(), srv.Client(), []byte("no links here")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
