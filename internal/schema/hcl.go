package schema

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// ignoreAll is the marker for lifecycle { ignore_changes = all }.
const ignoreAll = "*all*"

// registryPrefix is prepended to short provider sources such as
// hashicorp/azurerm to match the keys terraform uses in its schema output.
const registryPrefix = "registry.terraform.io/"

// entity is a resource or data block with everything the schema comparison
// needs: what the module sets, and where the block is.
type entity struct {
	address string // azurerm_subnet.this or data.azurerm_client_config.this
	typ     string // azurerm_subnet
	data    bool   // data source rather than resource
	file    string // relative to the module root
	line    int
	body    block
}

// block is the shape of a body: attribute names set, static nested blocks by
// type, dynamic nested blocks by label, and ignore_changes from lifecycle.
type block struct {
	attrs   map[string]bool
	static  map[string][]block
	dynamic map[string]*block
	ignore  []string
}

// module is everything parsed from one directory of .tf files.
type module struct {
	providers map[string]string // local name → source, e.g. azurerm → registry.terraform.io/hashicorp/azurerm
	entities  []entity
}

// parseDir reads every .tf file directly in dir. Paths in entities are made
// relative to root so nested modules report as modules/<name>/main.tf.
func parseDir(root, dir string) (*module, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	m := &module{providers: map[string]string{}}
	parser := hclparse.NewParser()
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".tf") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		f, diags := parser.ParseHCLFile(path)
		if diags.HasErrors() {
			return nil, fmt.Errorf("%s: %s", path, diags.Error())
		}
		body, ok := f.Body.(*hclsyntax.Body)
		if !ok {
			return nil, fmt.Errorf("%s: not native HCL syntax", path)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil, err
		}
		m.parseBody(rel, body)
	}
	return m, nil
}

func (m *module) parseBody(file string, body *hclsyntax.Body) {
	for _, b := range body.Blocks {
		switch {
		case b.Type == "terraform":
			m.parseRequiredProviders(b.Body)
		case (b.Type == "resource" || b.Type == "data") && len(b.Labels) == 2:
			e := entity{
				typ:  b.Labels[0],
				data: b.Type == "data",
				file: file,
				line: b.TypeRange.Start.Line,
				body: parseBlock(b.Body),
			}
			e.address = b.Labels[0] + "." + b.Labels[1]
			if e.data {
				e.address = "data." + e.address
			}
			m.entities = append(m.entities, e)
		}
	}
}

// parseRequiredProviders records source per local provider name from
// terraform { required_providers { name = { source = ... } } }. A missing
// source defaults to hashicorp/<name>, as terraform itself does.
func (m *module) parseRequiredProviders(body *hclsyntax.Body) {
	for _, b := range body.Blocks {
		if b.Type != "required_providers" {
			continue
		}
		attrs, _ := b.Body.JustAttributes()
		for name, attr := range attrs {
			val, _ := attr.Expr.Value(nil)
			source := "hashicorp/" + name
			if val.Type().IsObjectType() && val.Type().HasAttribute("source") {
				if s := val.GetAttr("source"); !s.IsNull() && s.Type() == cty.String {
					source = s.AsString()
				}
			}
			m.providers[name] = normalizeSource(source)
		}
	}
}

func normalizeSource(source string) string {
	if strings.Contains(source, "/") && !strings.HasPrefix(source, registryPrefix) {
		return registryPrefix + source
	}
	return source
}

// parseBlock records what a body sets. lifecycle is consumed for
// ignore_changes and not kept as a nested block; dynamic "x" { content {…} }
// counts as setting x and its content body is merged with any other dynamic
// block of the same label.
func parseBlock(body *hclsyntax.Body) block {
	b := block{
		attrs:   make(map[string]bool, len(body.Attributes)),
		static:  map[string][]block{},
		dynamic: map[string]*block{},
	}
	for name := range body.Attributes {
		b.attrs[name] = true
	}
	for _, nested := range body.Blocks {
		switch nested.Type {
		case "lifecycle":
			if attr, ok := nested.Body.Attributes["ignore_changes"]; ok {
				b.ignore = append(b.ignore, ignoreChanges(attr.Expr)...)
			}
		case "dynamic":
			if len(nested.Labels) != 1 {
				continue
			}
			label := nested.Labels[0]
			b.attrs[label] = true
			content := parseBlock(contentBody(nested.Body))
			if existing := b.dynamic[label]; existing != nil {
				existing.merge(content)
			} else {
				b.dynamic[label] = &content
			}
		default:
			b.static[nested.Type] = append(b.static[nested.Type], parseBlock(nested.Body))
		}
	}
	return b
}

// contentBody returns the content {} body of a dynamic block, or the dynamic
// block body itself when content is absent.
func contentBody(body *hclsyntax.Body) *hclsyntax.Body {
	for _, b := range body.Blocks {
		if b.Type == "content" {
			return b.Body
		}
	}
	return body
}

// ignoreChanges extracts attribute names from an ignore_changes expression.
// `all` yields ignoreAll. Names may be bare traversals (tags), strings, or a
// mix inside a list.
func ignoreChanges(expr hclsyntax.Expression) []string {
	switch e := expr.(type) {
	case *hclsyntax.TupleConsExpr:
		var names []string
		for _, item := range e.Exprs {
			names = append(names, ignoreChanges(item)...)
		}
		return names
	case *hclsyntax.ScopeTraversalExpr:
		name := e.Traversal.RootName()
		if name == "all" {
			return []string{ignoreAll}
		}
		return []string{name}
	case *hclsyntax.TemplateExpr:
		if len(e.Parts) == 1 {
			return ignoreChanges(e.Parts[0])
		}
	case *hclsyntax.LiteralValueExpr:
		if e.Val.Type() == cty.String {
			if s := e.Val.AsString(); s == "all" {
				return []string{ignoreAll}
			} else {
				return []string{s}
			}
		}
	}
	return nil
}

func (b *block) merge(other block) {
	for name := range other.attrs {
		b.attrs[name] = true
	}
	for name, blocks := range other.static {
		b.static[name] = append(b.static[name], blocks...)
	}
	for name, dyn := range other.dynamic {
		if existing := b.dynamic[name]; existing != nil {
			existing.merge(*dyn)
		} else {
			b.dynamic[name] = dyn
		}
	}
	b.ignore = append(b.ignore, other.ignore...)
}

// ignored reports whether name is covered by an ignore_changes list.
func ignored(ignore []string, name string) bool {
	return slices.ContainsFunc(ignore, func(i string) bool {
		return i == ignoreAll || strings.EqualFold(i, name)
	})
}
