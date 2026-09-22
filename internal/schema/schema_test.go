package schema

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var testSchema = &providerSchemas{Providers: map[string]*providerSchema{
	"registry.terraform.io/hashicorp/azurerm": {
		Resources: map[string]*entitySchema{
			"azurerm_thing": {Block: &blockSchema{
				Attributes: map[string]*attributeSchema{
					"id":       {Computed: true},
					"name":     {Required: true},
					"tags":     {Optional: true},
					"etag":     {Computed: true},
					"legacy":   {Optional: true, Deprecated: true},
					"location": {Optional: true, Computed: true},
				},
				BlockTypes: map[string]*blockTypeSchema{
					"timeouts": {Block: &blockSchema{}},
					"identity": {MinItems: 1, Block: &blockSchema{
						Attributes: map[string]*attributeSchema{
							"type":  {Required: true},
							"ids":   {Optional: true},
							"ident": {Computed: true},
						},
					}},
					"rule": {Block: &blockSchema{
						Attributes: map[string]*attributeSchema{
							"priority": {Required: true},
							"note":     {Optional: true},
						},
					}},
				},
			}},
		},
		DataSources: map[string]*entitySchema{
			"azurerm_thing": {Block: &blockSchema{
				Attributes: map[string]*attributeSchema{
					"name":   {Required: true},
					"filter": {Optional: true},
				},
			}},
		},
	},
}}

const providersTF = `terraform {
  required_providers {
    azurerm = { source = "hashicorp/azurerm" }
  }
}
`

// run writes main.tf into a temp module, parses it and compares against
// testSchema. It never invokes terraform.
func run(t *testing.T, mainTF string, noProviders bool, exclude ...string) []Finding {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{"main.tf": mainTF}
	if !noProviders {
		files["terraform.tf"] = providersTF
	}
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m, err := parseDir(dir, dir)
	if err != nil {
		t.Fatal(err)
	}
	return compare(m, testSchema, exclude)
}

// names flattens findings to "label name" strings for compact assertions.
func names(fs []Finding) []string {
	out := make([]string, 0, len(fs))
	for _, f := range fs {
		label := "optional"
		if f.Required {
			label = "required"
		}
		if f.Block {
			label += " block"
		}
		out = append(out, label+" "+f.Name)
	}
	return out
}

func TestCompare(t *testing.T) {
	t.Parallel()
	full := `resource "azurerm_thing" "this" {
  name     = "x"
  tags     = {}
  location = "we"
  identity {
    type = "SystemAssigned"
    ids  = []
  }
  rule {
    priority = 1
    note     = "n"
  }
}`
	tests := []struct {
		name        string
		mainTF      string
		exclude     []string
		noProviders bool     // omit terraform.tf
		want        []string // "label name", any order
	}{
		{
			name:   "everything set reports nothing",
			mainTF: full,
		},
		{
			name: "missing attributes and blocks with required flag from schema",
			mainTF: `resource "azurerm_thing" "this" {
}`,
			want: []string{"required name", "optional tags", "optional location", "required block identity", "optional block rule"},
		},
		{
			name: "id, computed-only, deprecated and timeouts are never reported",
			mainTF: `resource "azurerm_thing" "this" {
  name = "x"
  tags = {}
  location = "we"
  identity {
    type = "a"
    ids  = []
  }
  rule {
    priority = 1
    note     = "n"
  }
}`,
		},
		{
			name: "nested block attributes are reported with a dotted path",
			mainTF: `resource "azurerm_thing" "this" {
  name = "x"
  tags = {}
  location = "we"
  identity {
    type = "a"
  }
  rule {
    priority = 1
  }
}`,
			want: []string{"optional identity.ids", "optional rule.note"},
		},
		{
			name: "repeated static blocks are indexed",
			mainTF: `resource "azurerm_thing" "this" {
  name = "x"
  tags = {}
  location = "we"
  identity {
    type = "a"
    ids  = []
  }
  rule {
    priority = 1
    note     = "n"
  }
  rule {
    priority = 2
  }
}`,
			want: []string{"optional rule[1].note"},
		},
		{
			name: "dynamic block counts as set and its content is compared",
			mainTF: `resource "azurerm_thing" "this" {
  name = "x"
  tags = {}
  location = "we"
  identity {
    type = "a"
    ids  = []
  }
  dynamic "rule" {
    for_each = var.rules
    content {
      priority = rule.value.priority
    }
  }
}`,
			want: []string{"optional rule.note"},
		},
		{
			name: "two dynamic blocks of one label are merged before comparing",
			mainTF: `resource "azurerm_thing" "this" {
  name = "x"
  tags = {}
  location = "we"
  identity {
    type = "a"
    ids  = []
  }
  dynamic "rule" {
    for_each = var.a
    content {
      priority = 1
    }
  }
  dynamic "rule" {
    for_each = var.b
    content {
      note = "n"
    }
  }
}`,
		},
		{
			name: "ignore_changes hides attributes and blocks, inherited into nested blocks",
			mainTF: `resource "azurerm_thing" "this" {
  name = "x"
  location = "we"
  identity {
    type = "a"
  }
  lifecycle {
    ignore_changes = [tags, "rule", IDS]
  }
}`,
		},
		{
			name: "ignore_changes = all hides everything",
			mainTF: `resource "azurerm_thing" "this" {
  lifecycle {
    ignore_changes = all
  }
}`,
		},
		{
			name: "data source uses the data source schema",
			mainTF: `data "azurerm_thing" "this" {
  name = "x"
}`,
			want: []string{"optional filter"},
		},
		{
			name: "exclusions distinguish resource and data source types",
			mainTF: `resource "azurerm_thing" "a" {}
data "azurerm_thing" "b" {}`,
			exclude: []string{"azurerm_thing"},
			want:    []string{"required name", "optional filter"},
		},
		{
			name: "missing required_providers defaults to hashicorp/<prefix>",
			mainTF: `data "azurerm_thing" "this" {
  name = "x"
}`,
			noProviders: true,
			want:        []string{"optional filter"},
		},
		{
			name: "unknown provider or type is skipped",
			mainTF: `resource "random_pet" "p" {}
resource "azurerm_other" "o" {}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := names(run(t, tt.mainTF, tt.noProviders, tt.exclude...))
			if len(got) != len(tt.want) {
				t.Fatalf("got %d findings %v, want %d %v", len(got), got, len(tt.want), tt.want)
			}
			for _, w := range tt.want {
				found := false
				for _, g := range got {
					found = found || g == w
				}
				if !found {
					t.Errorf("missing %q in %v", w, got)
				}
			}
		})
	}
}

func TestFindingPosition(t *testing.T) {
	t.Parallel()
	fs := run(t, `# comment

resource "azurerm_thing" "here" {}`, false)
	if len(fs) == 0 {
		t.Fatal("expected findings")
	}
	if fs[0].File != "main.tf" || fs[0].Line != 3 || fs[0].Address != "azurerm_thing.here" {
		t.Fatalf("got %s:%d %s, want main.tf:3 azurerm_thing.here", fs[0].File, fs[0].Line, fs[0].Address)
	}
}

func TestParseRequiredProviders(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		tf   string
		want map[string]string
	}{
		{"explicit source is normalised", `terraform {
  required_providers {
    azurerm = { source = "hashicorp/azurerm", version = "~> 4.0" }
  }
}`,
			map[string]string{"azurerm": "registry.terraform.io/hashicorp/azurerm"}},
		{"missing source defaults to hashicorp", `terraform {
  required_providers {
    random = { version = "3.0" }
  }
}`,
			map[string]string{"random": "registry.terraform.io/hashicorp/random"}},
		{"full registry source kept", `terraform {
  required_providers {
    x = { source = "registry.terraform.io/acme/x" }
  }
}`,
			map[string]string{"x": "registry.terraform.io/acme/x"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "terraform.tf"), []byte(tt.tf), 0o644); err != nil {
				t.Fatal(err)
			}
			m, err := parseDir(dir, dir)
			if err != nil {
				t.Fatal(err)
			}
			for k, v := range tt.want {
				if m.providers[k] != v {
					t.Errorf("providers[%q] = %q, want %q", k, m.providers[k], v)
				}
			}
		})
	}
}

func TestParseDirInvalidHCL(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`resource "a" "b" {`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := parseDir(dir, dir); err == nil || !strings.Contains(err.Error(), "main.tf") {
		t.Fatalf("err = %v, want parse error naming main.tf", err)
	}
}

func TestReport(t *testing.T) {
	t.Parallel()
	got := Report([]Finding{
		{"main.tf", 18, "azurerm_virtual_network.this", "flow_timeout_in_minutes", false, false},
		{"main.tf", 18, "azurerm_virtual_network.this", "ddos_protection_plan", false, true},
		{"main.tf", 61, "azurerm_subnet.this", "delegation", true, true},
		{"modules/net/main.tf", 4, "data.azurerm_client_config.this", "name", true, false},
	})
	want := `main.tf: provider schema properties not set
line 18: azurerm_virtual_network.this
  optional: flow_timeout_in_minutes
  optional block: ddos_protection_plan
line 61: azurerm_subnet.this
  required block: delegation
modules/net/main.tf: provider schema properties not set
line 4: data.azurerm_client_config.this
  required: name`
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

// TestCheckFixture runs the real terraform binary against the example module,
// including provider download. Skipped in -short and when terraform is absent.
func TestCheckFixture(t *testing.T) {
	if testing.Short() {
		t.Skip("-short")
	}
	if _, err := exec.LookPath("terraform"); err != nil {
		t.Skip("terraform not in PATH")
	}
	dir := t.TempDir()
	entries, err := os.ReadDir("../../examples/module")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join("../../examples/module", e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, e.Name()), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// A submodule with a deliberately bare resource: the example root module is
	// complete, so this is what proves modules/* are discovered and reported
	// with paths relative to the root.
	sub := filepath.Join(dir, "modules", "bare")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, src := range map[string]string{
		"terraform.tf": providersTF,
		"main.tf": `resource "azurerm_resource_group" "this" {
  name     = "rg"
  location = "westeurope"
}`,
	} {
		if err := os.WriteFile(filepath.Join(sub, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	findings, err := Check(context.Background(), dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	report := Report(findings)
	t.Logf("findings:\n%s", report)

	for _, f := range findings {
		if f.File != "modules/bare/main.tf" {
			t.Errorf("root module reported %s %s; the example module must be complete", f.Address, f.Name)
		}
	}
	for _, want := range []string{
		"modules/bare/main.tf: provider schema properties not set",
		"line 1: azurerm_resource_group.this",
		"  optional: tags",
		"  optional: managed_by",
	} {
		if !strings.Contains(report, want) {
			t.Errorf("report lacks %q", want)
		}
	}

	// Nothing may be left behind in either module.
	for _, d := range []string{dir, sub} {
		for _, name := range []string{".terraform", ".terraform.lock.hcl", "terraform.tfstate"} {
			if _, err := os.Stat(filepath.Join(d, name)); err == nil {
				t.Errorf("%s left behind in %s", name, d)
			}
		}
	}
}
