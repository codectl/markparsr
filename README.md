# shapr

Go package keeping Terraform modules in shape: docs generated, schema verified.

One renderer, two modes: `generate` writes the README block, `check` verifies it. Same code, so they can never disagree. Add `WithSchema()` and every resource is also held against its provider schema.

## Why shapr?

Terraform modules evolve rapidly. Documentation lags behind, and providers grow properties the module never exposes. Catching either used to take separate tools with separate parsers and separate opinions.

`shapr helps you:`

Generate the README block from Terraform sources.

Fail pull requests on documentation drift, as a Go test wired up as a required check.

Report attributes and blocks the provider schema offers that the module never sets.

Enforce your own conventions: required files, extra sections, live URLs.

## Installation

`go get github.com/codectl/shapr`

## Usage

Generate or update the docs of a module:

`go run github.com/codectl/shapr/cmd/shapr generate .`

Check from a Go test, run by CI on every pull request:

```go
func TestModule(t *testing.T) {
	v, err := shapr.New(
		shapr.WithModule(".."),
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
```

A failing check lists every change in words, with line numbers and the fix:

```
README.md: generated docs out of date; run: shapr generate .
line 14: differs:
  README:  ## Providerss
  sources: ## Providers
line 27: in README, not generated from Terraform sources:
  - - [azurerm_bogus.this](https://example.com) (resource)
line 32: missing from README, generated from Terraform sources:
  + - [azurerm_virtual_network.this](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/virtual_network) (resource)
main.tf: provider schema properties not set
line 2: azurerm_virtual_network.this
  optional block: ddos_protection_plan
  optional: private_endpoint_vnet_policies
line 41: azurerm_subnet.this
  optional: sharing_scope
modules/dns/main.tf: provider schema properties not set
line 4: data.azurerm_client_config.this
  optional: ...
```

## Features

`Generated Block`

Renders `terraform-docs markdown document --hide modules` byte-identical to the CLI, pinned in `go.mod`.

Injects between `<!-- BEGIN_TF_DOCS -->` and `<!-- END_TF_DOCS -->`; prose outside is never touched.

`Provider Schema`

Compares every resource and data source, including those under `modules/*`, against `terraform providers schema -json`.

Skips `id`, computed-only, deprecated, `timeouts`, and anything in `lifecycle.ignore_changes`.

Follows nested blocks, repeated blocks, and `dynamic` blocks.

Runs `terraform init` with a temporary data directory and leaves the module untouched.

`Module Conventions`

Ensures `README.md`, `variables.tf`, `outputs.tf`, `terraform.tf` and any additional files exist and are non-empty.

Requires additional `## Section` headings to be present.

Validates URLs in the README respond with 200.

## Configuration

`WithModule(dir)`: Module directory; required. Its `README.md` is validated.

`WithSections(names...)`: Additional headings that must exist.

`WithFiles(names...)`: Additional files, relative to the module, that must exist and be non-empty.

`WithSchema()`: Enable the provider schema check. Needs the `terraform` binary.

`WithSchemaExclusions(types...)`: Skip types in the schema check; resource types bare (`azurerm_subnet`), data sources prefixed (`data.azurerm_subnet`).

`shapr check -schema -exclude azurerm_subnet,data.azurerm_client_config .` does the same from the CLI.

### Notes

shapr embeds terraform-docs as a library and requires Go 1.25.

CI never writes. The check fails and tells the developer to regenerate; it does not commit to the pull request.

The schema check downloads providers; give CI the `terraform` binary and network access.

## Contributors

We welcome contributions from the community! Whether it's reporting a bug, suggesting a new feature, or submitting a pull request, your input is highly valued.
