package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func tagResourceSchema(description, title string, metadata, spec map[string]schema.Attribute) schema.Schema {
	return schema.Schema{
		MarkdownDescription: description,
		Attributes: map[string]schema.Attribute{
			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: title + " metadata.",
				Required:            true,
				Attributes:          metadata,
			},
			"spec": schema.SingleNestedAttribute{
				MarkdownDescription: title + " specification.",
				Required:            true,
				Attributes:          spec,
			},
		},
	}
}

func tagValuesAttribute() schema.ListAttribute {
	return schema.ListAttribute{
		MarkdownDescription: "List of values for this tag.",
		Required:            true,
		ElementType:         types.StringType,
	}
}

func tagsMapAttribute(objectName string) schema.MapAttribute {
	return schema.MapAttribute{
		MarkdownDescription: "All tags to set on the " + objectName + ". This map is authoritative: every apply writes " +
			"exactly these tags, so any tag not listed here is removed. Only these keys are tracked on refresh — " +
			"tags added under other keys outside Terraform do not show up as drift.",
		ElementType: types.ListType{ElemType: types.StringType},
		Required:    true,
	}
}

func trackedTagValues(ctx context.Context, tags map[string][]string, key string, tracked types.List) (values types.List, found bool, diags diag.Diagnostics) {
	vals, ok := tags[key]
	if !ok {
		// The API cannot represent a tag with no values — a PUT carrying `{"k": []}` comes back with
		// `"tags": {}` — so for a resource whose tracked value list is empty, "absent" is that same state
		// and must not be read as a deletion, or the resource is destroyed and recreated on every apply.
		// On import there is no prior state (values is null), so an absent key really is missing.
		declaredEmpty := !tracked.IsNull() && len(tracked.Elements()) == 0
		if !declaredEmpty {
			return types.ListNull(types.StringType), false, nil
		}
		vals = []string{}
	}
	values, diags = types.ListValueFrom(ctx, types.StringType, vals)
	return values, true, diags
}
