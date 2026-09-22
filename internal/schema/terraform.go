package schema

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const lockFile = ".terraform.lock.hcl"

// providerSchemas is the subset of `terraform providers schema -json` shapr
// reads, keyed by provider source.
type providerSchemas struct {
	Providers map[string]*providerSchema `json:"provider_schemas"`
}

type providerSchema struct {
	Resources   map[string]*entitySchema `json:"resource_schemas"`
	DataSources map[string]*entitySchema `json:"data_source_schemas"`
}

type entitySchema struct {
	Block *blockSchema `json:"block"`
}

type blockSchema struct {
	Attributes map[string]*attributeSchema `json:"attributes"`
	BlockTypes map[string]*blockTypeSchema `json:"block_types"`
}

type attributeSchema struct {
	Required   bool `json:"required"`
	Optional   bool `json:"optional"`
	Computed   bool `json:"computed"`
	Deprecated bool `json:"deprecated"`
}

type blockTypeSchema struct {
	MinItems   int          `json:"min_items"`
	Block      *blockSchema `json:"block"`
	Deprecated bool         `json:"deprecated"`
}

// loadSchemas runs terraform init and providers schema in dir without leaving
// anything behind: the working directory goes to a temp TF_DATA_DIR and the
// lock file is removed afterwards unless the module already had one.
func loadSchemas(ctx context.Context, dir string) (*providerSchemas, error) {
	if _, err := exec.LookPath("terraform"); err != nil {
		return nil, errors.New("terraform binary not found in PATH")
	}
	dataDir, err := os.MkdirTemp("", "shapr-tf-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dataDir)

	lock := filepath.Join(dir, lockFile)
	if _, err := os.Stat(lock); errors.Is(err, os.ErrNotExist) {
		defer os.Remove(lock)
	}

	env := append(os.Environ(), "TF_DATA_DIR="+dataDir, "TF_IN_AUTOMATION=1")

	init := exec.CommandContext(ctx, "terraform", "init", "-backend=false", "-input=false", "-no-color")
	init.Dir, init.Env = dir, env
	if out, err := init.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("terraform init in %s: %w\n%s", dir, err, out)
	}

	dump := exec.CommandContext(ctx, "terraform", "providers", "schema", "-json")
	dump.Dir, dump.Env = dir, env
	out, err := dump.Output()
	if err != nil {
		return nil, fmt.Errorf("terraform providers schema in %s: %w", dir, err)
	}
	var schemas providerSchemas
	if err := json.Unmarshal(out, &schemas); err != nil {
		return nil, fmt.Errorf("terraform providers schema in %s: %w", dir, err)
	}
	return &schemas, nil
}
