// Package shapr keeps Terraform modules in shape: docs generated, schema
// verified.
//
// Generate writes the terraform-docs block of a module README. Validate
// renders the same block in memory and fails on any drift, applies the
// module conventions terraform-docs does not cover (required files,
// additional section headings, URL liveness) and, with WithSchema, compares
// every resource and data source against the provider schema. Module
// repositories run Validate from a Go test wired up as a required check.
package shapr

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/codectl/shapr/internal/check"
	"github.com/codectl/shapr/internal/diff"
	"github.com/codectl/shapr/internal/schema"
	"github.com/codectl/shapr/internal/tfdocs"
)

const readmeName = "README.md"

var requiredFiles = []string{readmeName, "variables.tf", "outputs.tf", "terraform.tf"}

var httpClient = &http.Client{Timeout: 10 * time.Second}

type Validator struct {
	dir           string
	sections      []string
	files         []string
	schema        bool
	schemaExclude []string
}

type Option func(*Validator)

func WithModule(dir string) Option {
	return func(v *Validator) { v.dir = dir }
}

func WithSections(names ...string) Option {
	return func(v *Validator) { v.sections = append(v.sections, names...) }
}

func WithFiles(names ...string) Option {
	return func(v *Validator) { v.files = append(v.files, names...) }
}

// WithSchema compares every resource and data source, including those under
// modules/*, against the provider schema and reports attributes and blocks
// the module never sets. Needs the terraform binary.
func WithSchema() Option {
	return func(v *Validator) { v.schema = true }
}

// WithSchemaExclusions skips types in the schema check: resource types bare
// ("azurerm_subnet"), data source types prefixed ("data.azurerm_subnet").
func WithSchemaExclusions(types ...string) Option {
	return func(v *Validator) { v.schemaExclude = append(v.schemaExclude, types...) }
}

func New(opts ...Option) (*Validator, error) {
	v := &Validator{}
	for _, opt := range opts {
		opt(v)
	}
	if v.dir == "" {
		return nil, errors.New("shapr: WithModule is required")
	}
	dir, err := filepath.Abs(v.dir)
	if err != nil {
		return nil, fmt.Errorf("shapr: %w", err)
	}
	v.dir = dir
	return v, nil
}

func (v *Validator) Validate(ctx context.Context) error {
	var errs []error
	if err := check.Files(v.dir, requiredFiles...); err != nil {
		errs = append(errs, err)
	}
	if err := check.Files(v.dir, v.files...); err != nil {
		errs = append(errs, err)
	}

	if readme, err := os.ReadFile(filepath.Join(v.dir, readmeName)); err == nil {
		if err := v.drift(readme); err != nil {
			errs = append(errs, err)
		}
		if err := check.Sections(readme, v.sections...); err != nil {
			errs = append(errs, err)
		}
		if err := check.URLs(ctx, httpClient, readme); err != nil {
			errs = append(errs, err)
		}
	}

	if v.schema {
		findings, err := schema.Check(ctx, v.dir, v.schemaExclude)
		if err != nil {
			errs = append(errs, err)
		} else if len(findings) > 0 {
			errs = append(errs, errors.New(schema.Report(findings)))
		}
	}
	return errors.Join(errs...)
}

func Generate(dir string) error {
	content, err := tfdocs.Render(dir)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, readmeName)
	readme, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	out, err := tfdocs.Inject(readme, content)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return os.WriteFile(path, out, 0o644)
}

func (v *Validator) drift(readme []byte) error {
	want, err := tfdocs.Render(v.dir)
	if err != nil {
		return err
	}
	have, err := tfdocs.Block(readme)
	if err != nil {
		return fmt.Errorf("%s: %w; run: shapr generate %s", readmeName, err, v.dir)
	}

	offset := bytes.Count(readme[:bytes.Index(readme, tfdocs.Begin)], []byte("\n")) + 2
	report := diff.Report(have, want, offset)
	if report == "" {
		return nil
	}
	return fmt.Errorf("%s: generated docs out of date; run: shapr generate %s\n%s", readmeName, v.dir, report)
}
