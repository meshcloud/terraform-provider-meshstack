package provider

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
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
	return setdefault.StaticValue(types.SetValueMust(nestedObjectToObjectType(nested), nil))
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
				Default:             stringdefault.StaticString(client.MeshObjectKind.ProjectRole),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

// meshUuidRefAttribute builds the attributes for a user-supplied reference composed of a fixed
// `kind` discriminator plus a `uuid`. `kind` is optional and defaults to (and is validated against)
// the single fixed kind; `uuid` is left Optional+Computed so a whole computed `.ref` object — whose
// uuid is unknown until apply — can be assigned and so the backend can default an omitted ref. Pair
// it with meshUuidRefValidators so an explicitly-provided ref still has to carry its uuid.
//
// It intentionally does not fit refs with bespoke schemas: a discriminated uuid-or-name target_ref,
// a RequiresReplace platform_ref, or a uuid-only version_ref. Computed-only output refs (a resource's
// own `.ref`) use meshUuidRefOutputAttribute instead.
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

// meshUuidRefValidators guards a meshUuidRefAttribute reference: a ref object that is provided must
// carry its uuid. Assigning a whole computed `.ref` (whose uuid is unknown until apply) still
// passes — AlsoRequires treats an unknown value as configured — so only an explicitly omitted uuid
// is rejected, at plan time, instead of only failing later against the backend.
func meshUuidRefValidators() []validator.Object {
	return []validator.Object{
		objectvalidator.AlsoRequires(path.MatchRelative().AtName("uuid")),
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

// previewDisclaimer returns a standard MarkdownDescription note for resources and data sources
// that use a preview API. Append this to the MarkdownDescription of any preview resource.
func previewDisclaimer() string {
	return "\n\n~> **Preview:** This resource is in preview. " +
		"Breaking changes are possible without prior notice due to changes in the underlying [meshStack preview API](https://docs.meshcloud.io/api/technical-specifications#preview-endpoints) or due to changes in this provider. " +
		"Please ensure you are running the latest version of the provider and report any bugs via [GitHub issues](https://github.com/meshcloud/terraform-provider-meshstack/issues) " +
		"or via support@meshcloud.io."
}
