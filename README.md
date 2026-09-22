# markparsr

Generates your Terraform module docs and fails the build when they drift.

One renderer, two modes: `generate` writes the README block, `check` verifies it. Same code, so they can never disagree.

## Why markparsr?

Terraform modules evolve rapidly and documentation lags behind. Keeping the README current used to need two tools: the terraform-docs CLI to write it, and a separate validator that re-parsed HCL and scraped markdown to judge it. Two parsers, two opinions, and an unpinned CLI in between.

`markparsr helps you:`

Generate the README block from Terraform sources.

Fail pull requests on any drift, as a Go test wired up as a required check.

Pin one terraform-docs version in `go.mod` for generating and checking alike.

Enforce your own conventions: required files, extra sections, live URLs.

## Installation

`go get github.com/codectl/markparsr`

## Usage

See the [examples/](examples/) directory for a sample module and validator test.

Generate or update the docs of a module:

`go run github.com/codectl/markparsr/cmd/markparsr generate .`

Check from a Go test, run by CI on every pull request:

```go
func TestReadme(t *testing.T) {
	v, err := markparsr.New(
		markparsr.WithModule(".."),
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
```

A failing check lists every change in words, with README line numbers and
the fix:

```
README.md: generated docs out of date; run: markparsr generate .
line 14: differs:
  README:  ## Providerss
  sources: ## Providers
line 27: in README, not generated from Terraform sources:
  - - [azurerm_bogus.this](https://example.com) (resource)
line 32: missing from README, generated from Terraform sources:
  + - [azurerm_virtual_network.this](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/virtual_network) (resource)
line 146: differs:
  README:  Description: stale text
  sources: Description: default azure region to be used.
```

## Configuration

`WithModule(dir)`: Module directory; required. Its `README.md` is validated.

`WithSections(names...)`: Additional headings that must exist.

`WithFiles(names...)`: Additional files, relative to the module, that must exist and be non-empty.

### Notes

markparsr embeds terraform-docs as a library and requires Go 1.25.

CI never writes. The check fails and tells the developer to regenerate; it does not commit to the pull request.

## Contributors

We welcome contributions from the community! Whether it's reporting a bug, suggesting a new feature, or submitting a pull request, your input is highly valued.
