package provider

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// nestedObjectToObjectType converts a schema.NestedAttributeObject into a types.ObjectType
// by extracting the attr.Type from each attribute.
func nestedObjectToObjectType(nested schema.NestedAttributeObject) types.ObjectType {
	attrTypes := make(map[string]attr.Type, len(nested.Attributes))
	for name, attribute := range nested.Attributes {
		attrTypes[name] = attribute.GetType()
	}
	return types.ObjectType{AttrTypes: attrTypes}
}

// emptySetDefault returns an empty set whose element type is derived from the given
// schema.NestedAttributeObject as a default value.
func emptySetDefault(nested schema.NestedAttributeObject) defaults.Set {
	return setdefault.StaticValue(types.SetValueMust(nestedObjectToObjectType(nested), []attr.Value{}))
}

// meshProjectRoleAttribute returns a schema attribute for meshProject role references.
// This is used across multiple resources (landingzone, platform) to maintain consistency.
func meshProjectRoleAttribute(computed bool) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "the meshProject role",
		Required:            true,
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Computed:            computed,
				Required:            !computed,
				MarkdownDescription: "The identifier of the meshProjectRole",
			},
			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshProjectRole`.",
				Computed:            true,
				Default:             stringdefault.StaticString("meshProjectRole"),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func meshBuildingBlockDefinitionRefAttribute(computed bool) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"kind": schema.StringAttribute{
			MarkdownDescription: "meshObject type, always `meshBuildingBlockDefinition`.",
			Computed:            true,
			Default:             stringdefault.StaticString("meshBuildingBlockDefinition"),
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"uuid": schema.StringAttribute{
			MarkdownDescription: "UUID of the building block.",
			Computed:            computed,
			Required:            !computed,
		},
	}
}

// TODO reuse this at all other places where UUID refs.
func meshUuidRefAttribute(kind string) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"kind": schema.StringAttribute{
			MarkdownDescription: "meshObject type, always `" + kind + "`.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(kind),
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			Validators: []validator.String{
				stringvalidator.OneOf(kind),
			},
		},
		"uuid": schema.StringAttribute{
			MarkdownDescription: "UUID of the " + kind + ".",
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
	}
}

func meshUuidRefOutputAttribute(kind string) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"kind": schema.StringAttribute{
			MarkdownDescription: "meshObject type, always `" + kind + "`.",
			Computed:            true,
			Default:             stringdefault.StaticString(kind),
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"uuid": schema.StringAttribute{
			MarkdownDescription: "UUID of the " + kind + ".",
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
	}
}

func tenantTagsAttribute() schema.SingleNestedAttribute {
	tagMappersNested := schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"key": schema.StringAttribute{
				MarkdownDescription: "Key for the tag mapper",
				Required:            true,
			},
			"value_pattern": schema.StringAttribute{
				MarkdownDescription: "Value pattern for the tag mapper",
				Required:            true,
			},
		},
	}

	return schema.SingleNestedAttribute{
		MarkdownDescription: "Tenant tags configuration",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"namespace_prefix": schema.StringAttribute{
				MarkdownDescription: "This is the prefix for all labels created by meshStack. It helps to keep track of which labels are managed by meshStack. It is recommended to let this prefix end with a delimiter like an underscore.",
				Required:            true,
			},
			"tag_mappers": schema.SetNestedAttribute{
				MarkdownDescription: "Set of tag mappers for tenant tags",
				Optional:            true,
				Computed:            true,
				Default:             emptySetDefault(tagMappersNested),
				NestedObject:        tagMappersNested,
			},
		},
	}
}
