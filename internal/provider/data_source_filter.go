package provider

import (
	"fmt"
	"maps"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// asFilter lets configuration set the given attributes of a computed block, each narrowing the search.
func asFilter(attribute schema.Attribute, filterAttributes ...string) schema.SingleNestedAttribute {
	block, ok := attribute.(schema.SingleNestedAttribute)
	if !ok {
		panic(fmt.Sprintf("a filter block must be a SingleNestedAttribute, got %T", attribute))
	}
	block.Optional = true
	block.Attributes = maps.Clone(block.Attributes)
	for _, name := range filterAttributes {
		filter, ok := block.Attributes[name].(schema.StringAttribute)
		if !ok {
			panic(fmt.Sprintf("filter attribute %s must be a StringAttribute, got %T", name, block.Attributes[name]))
		}
		filter.Optional = true
		filter.MarkdownDescription += " Set it to narrow the lookup."
		block.Attributes[name] = filter
	}
	return block
}

// onlyMatch returns the one candidate the filter matches, and reports an error when none or several do.
func onlyMatch[T any](kind string, filter fmt.Stringer, candidates []T, matches func(T) bool, describe func(T) string, diags *diag.Diagnostics) (match T) {
	var found []T
	for _, candidate := range candidates {
		if matches(candidate) {
			found = append(found, candidate)
		}
	}
	switch len(found) {
	case 1:
		return found[0]
	case 0:
		diags.AddError(capitalize(kind)+" not found",
			fmt.Sprintf("No %s that you can see matches %s.", kind, filter))
	default:
		described := make([]string, len(found))
		for i, candidate := range found {
			described[i] = describe(candidate)
		}
		diags.AddError(capitalize(kind)+" is ambiguous",
			fmt.Sprintf("%d %ss match %s: %s. Set `metadata.uuid` to pick exactly one.",
				len(found), kind, filter, strings.Join(described, ", ")))
	}
	return
}
