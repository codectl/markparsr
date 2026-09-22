// Command shapr generates or checks Terraform module documentation and,
// optionally, the module against its provider schema.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/codectl/shapr"
)

const usage = `usage:
  shapr generate <dir>...
  shapr check [-schema] [-exclude type,...] <dir>...`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("%s", usage)
	}
	cmd := args[0]
	var op func(string) error
	switch cmd {
	case "generate":
		op = shapr.Generate
		args = args[1:]
	case "check":
		fs := flag.NewFlagSet("check", flag.ContinueOnError)
		withSchema := fs.Bool("schema", false, "compare resources against the provider schema (needs terraform)")
		exclude := fs.String("exclude", "", "comma-separated types to skip in the schema check; data sources as data.<type>")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		args = fs.Args()
		op = func(dir string) error {
			opts := []shapr.Option{shapr.WithModule(dir)}
			if *withSchema {
				opts = append(opts, shapr.WithSchema())
			}
			if *exclude != "" {
				opts = append(opts, shapr.WithSchemaExclusions(strings.Split(*exclude, ",")...))
			}
			v, err := shapr.New(opts...)
			if err != nil {
				return err
			}
			return v.Validate(context.Background())
		}
	default:
		return fmt.Errorf("unknown command %q\n%s", cmd, usage)
	}
	if len(args) == 0 {
		return fmt.Errorf("%s", usage)
	}

	failed := false
	for _, dir := range args {
		if err := op(dir); err != nil {
			failed = true
			fmt.Fprintf(os.Stderr, "%s: %v\n", dir, err)
		}
	}
	if failed {
		return fmt.Errorf("shapr %s failed", cmd)
	}
	return nil
}
