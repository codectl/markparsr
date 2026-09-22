// Package tfdocs is the single boundary to terraform-docs. It renders a
// Terraform module the way `terraform-docs markdown document --hide modules`
// does and injects the result between the BEGIN_TF_DOCS/END_TF_DOCS markers.
package tfdocs

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/terraform-docs/terraform-docs/format"
	"github.com/terraform-docs/terraform-docs/print"
	"github.com/terraform-docs/terraform-docs/terraform"
)

var (
	Begin = []byte(print.OutputBeginComment)
	End   = []byte(print.OutputEndComment)
)

var ErrNoBlock = errors.New("readme has no BEGIN_TF_DOCS/END_TF_DOCS markers")

func Render(dir string) ([]byte, error) {
	cfg := print.DefaultConfig()
	cfg.ModuleRoot = dir
	cfg.Formatter = "markdown document"
	cfg.Sections.Hide = []string{"modules"}
	cfg.Parse()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("tfdocs: config: %w", err)
	}

	module, err := terraform.LoadWithOptions(cfg)
	if err != nil {
		return nil, fmt.Errorf("tfdocs: load %s: %w", dir, err)
	}
	formatter, err := format.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("tfdocs: formatter: %w", err)
	}
	if err := formatter.Generate(module); err != nil {
		return nil, fmt.Errorf("tfdocs: generate: %w", err)
	}
	out, err := formatter.Render("")
	if err != nil {
		return nil, fmt.Errorf("tfdocs: render: %w", err)
	}
	return []byte(out), nil
}

func Block(readme []byte) ([]byte, error) {
	begin := bytes.Index(readme, Begin)
	end := bytes.Index(readme, End)
	switch {
	case begin < 0 && end < 0:
		return nil, ErrNoBlock
	case begin < 0:
		return nil, errors.New("readme has END_TF_DOCS but no BEGIN_TF_DOCS marker")
	case end < 0:
		return nil, errors.New("readme has BEGIN_TF_DOCS but no END_TF_DOCS marker")
	case end < begin:
		return nil, errors.New("readme has END_TF_DOCS before BEGIN_TF_DOCS marker")
	}
	start := begin + len(Begin)
	if start < end && readme[start] == '\n' {
		start++
	}
	if end > start && readme[end-1] == '\n' {
		end--
	}
	return readme[start:end], nil
}

func Inject(readme, content []byte) ([]byte, error) {
	block := wrap(content)
	if len(readme) == 0 {
		return block, nil
	}
	if _, err := Block(readme); err != nil {
		if errors.Is(err, ErrNoBlock) {
			return append(append(readme, '\n'), block...), nil
		}
		return nil, err
	}
	begin := bytes.Index(readme, Begin)
	end := bytes.Index(readme, End) + len(End)

	out := make([]byte, 0, len(readme)-(end-begin)+len(block))
	out = append(out, readme[:begin]...)
	out = append(out, block...)
	out = append(out, readme[end:]...)
	return out, nil
}

func wrap(content []byte) []byte {
	out := make([]byte, 0, len(Begin)+1+len(content)+1+len(End))
	out = append(out, Begin...)
	out = append(out, '\n')
	out = append(out, content...)
	out = append(out, '\n')
	out = append(out, End...)
	return out
}
