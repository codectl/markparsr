package schema

import (
	"fmt"
	"strings"
)

// Finding is one attribute or block the provider schema offers that the
// module never sets.
type Finding struct {
	File     string // relative to the module root
	Line     int    // of the resource or data block
	Address  string // azurerm_subnet.this or data.azurerm_client_config.this
	Name     string // attribute or block name, dotted for nested: identity.type
	Required bool
	Block    bool
}

// compare walks every entity of m against schemas. Entities whose provider,
// or whose type, has no schema are skipped: terraform init would have failed
// on an unknown provider, and an unknown type is terraform validate's job.
func compare(m *module, schemas *providerSchemas, exclude []string) []Finding {
	var findings []Finding
	for _, e := range m.entities {
		if excluded(e, exclude) {
			continue
		}
		prefix := strings.SplitN(e.typ, "_", 2)[0]
		source, ok := m.providers[prefix]
		if !ok {
			// No required_providers entry: terraform assumes hashicorp/<prefix>.
			source = normalizeSource("hashicorp/" + prefix)
		}
		ps := schemas.Providers[source]
		if ps == nil {
			continue
		}
		var es *entitySchema
		if e.data {
			es = ps.DataSources[e.typ]
		} else {
			es = ps.Resources[e.typ]
		}
		if es == nil || es.Block == nil {
			continue
		}
		findings = compareBlock(findings, e, "", e.body, es.Block, nil)
	}
	return findings
}

// excluded matches resource types bare and data source types with a data.
// prefix, so azurerm_subnet and data.azurerm_subnet are distinct.
func excluded(e entity, exclude []string) bool {
	key := e.typ
	if e.data {
		key = "data." + key
	}
	for _, x := range exclude {
		if x == key {
			return true
		}
	}
	return false
}

// compareBlock appends a Finding for each attribute and nested block in
// schema that b does not set, then recurses into nested blocks b does set.
// ignore accumulates lifecycle.ignore_changes from the entity down.
func compareBlock(findings []Finding, e entity, path string, b block, schema *blockSchema, ignore []string) []Finding {
	ignore = append(ignore[:len(ignore):len(ignore)], b.ignore...)

	for name, attr := range schema.Attributes {
		switch {
		case name == "id",
			attr.Deprecated,
			attr.Computed && !attr.Optional && !attr.Required,
			ignored(ignore, name),
			b.attrs[name]:
			continue
		}
		findings = append(findings, Finding{e.file, e.line, e.address, path + name, attr.Required, false})
	}

	for name, bt := range schema.BlockTypes {
		if name == "timeouts" || bt.Deprecated || ignored(ignore, name) {
			continue
		}
		static, dynamic := b.static[name], b.dynamic[name]
		if len(static) == 0 && dynamic == nil {
			findings = append(findings, Finding{e.file, e.line, e.address, path + name, bt.MinItems > 0, true})
			continue
		}
		if bt.Block == nil {
			continue
		}
		for i, nested := range static {
			nestedPath := path + name + "."
			if len(static) > 1 {
				nestedPath = fmt.Sprintf("%s%s[%d].", path, name, i)
			}
			findings = compareBlock(findings, e, nestedPath, nested, bt.Block, ignore)
		}
		if dynamic != nil {
			findings = compareBlock(findings, e, path+name+".", *dynamic, bt.Block, ignore)
		}
	}
	return findings
}
