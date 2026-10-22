package provider

import (
	"fmt"
	"maps"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
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
