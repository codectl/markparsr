package diff

import "testing"

func TestReportEqual(t *testing.T) {
	t.Parallel()
	if got := Report([]byte("a\nb"), []byte("a\nb"), 1); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestReportClassifiesEveryChange(t *testing.T) {
	t.Parallel()
	// Block starts at README line 6. Three faults: a heading typo, a resource
	// only in the README, a resource only in the sources.
	have := "## Requirements\n\n## Providerss\n\n## Resources\n\n- a\n- stale\n- c\n"
	want := "## Requirements\n\n## Providers\n\n## Resources\n\n- a\n- c\n- new\n"
	got := Report([]byte(have), []byte(want), 6)
	wantOut := `line 8: differs:
  README:  ## Providerss
  sources: ## Providers
line 13: in README, not generated from Terraform sources:
  - - stale
line 15: missing from README, generated from Terraform sources:
  + - new`
	if got != wantOut {
		t.Fatalf("got:\n%s\nwant:\n%s", got, wantOut)
	}
}
