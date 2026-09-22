// Command markparsr generates or checks the terraform-docs block of one or
// more Terraform module READMEs.
//
//	markparsr generate <dir>...
//	markparsr check <dir>...
//
// check exits 1 on drift, missing required files, or dead URLs.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/codectl/markparsr"
)

const usage = "usage: markparsr generate|check <dir>..."

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("%s", usage)
	}
	var op func(string) error
	switch args[0] {
	case "generate":
		op = markparsr.Generate
	case "check":
		op = checkModule
	default:
		return fmt.Errorf("unknown command %q\n%s", args[0], usage)
	}

	failed := false
	for _, dir := range args[1:] {
		if err := op(dir); err != nil {
			failed = true
			fmt.Fprintf(os.Stderr, "%s: %v\n", dir, err)
		}
	}
	if failed {
		return fmt.Errorf("markparsr %s failed", args[0])
	}
	return nil
}

func checkModule(dir string) error {
	v, err := markparsr.New(markparsr.WithModule(dir))
	if err != nil {
		return err
	}
	return v.Validate(context.Background())
}
