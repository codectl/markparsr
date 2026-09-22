// Package markparsr generates and verifies the terraform-docs block of a
// Terraform module README with one codepath.
//
// Generate writes the block; Validate renders the same block in memory and
// fails on any drift, then applies the checks terraform-docs does not cover:
// required files, additional section headings, and URL liveness. Module
// repositories run Validate from a Go test wired up as a required check.
package markparsr

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/codectl/markparsr/internal/check"
	"github.com/codectl/markparsr/internal/diff"
	"github.com/codectl/markparsr/internal/tfdocs"
)

const readmeName = "README.md"

var requiredFiles = []string{readmeName, "variables.tf", "outputs.tf", "terraform.tf"}

var httpClient = &http.Client{Timeout: 10 * time.Second}

type Validator struct {
	dir      string
	sections []string
	files    []string
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

func New(opts ...Option) (*Validator, error) {
	v := &Validator{}
	for _, opt := range opts {
		opt(v)
	}
	if v.dir == "" {
		return nil, errors.New("markparsr: WithModule is required")
	}
	dir, err := filepath.Abs(v.dir)
	if err != nil {
		return nil, fmt.Errorf("markparsr: %w", err)
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

	readme, err := os.ReadFile(filepath.Join(v.dir, readmeName))
	if err != nil {
		return errors.Join(errs...)
	}

	if err := v.drift(readme); err != nil {
		errs = append(errs, err)
	}
	if err := check.Sections(readme, v.sections...); err != nil {
		errs = append(errs, err)
	}
	if err := check.URLs(ctx, httpClient, readme); err != nil {
		errs = append(errs, err)
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
		return fmt.Errorf("%s: %w; run: markparsr generate %s", readmeName, err, v.dir)
	}

	offset := bytes.Count(readme[:bytes.Index(readme, tfdocs.Begin)], []byte("\n")) + 2
	report := diff.Report(have, want, offset)
	if report == "" {
		return nil
	}
	return fmt.Errorf("%s: generated docs out of date; run: markparsr generate %s\n%s", readmeName, v.dir, report)
}
