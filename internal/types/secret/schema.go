package secret

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/internal/util/maps"
)

type AttributeSchemaOptions struct {
	MarkdownDescription string
	Optional            bool
}

func AttributeSchema(opts AttributeSchemaOptions) (result schema.SingleNestedAttribute) {
	return schema.SingleNestedAttribute{
		MarkdownDescription: opts.MarkdownDescription,
		Optional:            opts.Optional,
		Required:            !opts.Optional,
		CustomType:          typeImpl{ObjectType: types.ObjectType{AttrTypes: secretAttributeTypes}},
		Attributes:          secretAttributes(opts),
	}
}

var (
	secretAttributes = func(opts AttributeSchemaOptions) map[string]schema.Attribute {
		return map[string]schema.Attribute{
			"value": schema.StringAttribute{
				MarkdownDescription: opts.MarkdownDescription,
				Required:            true,
				Sensitive:           true,
				WriteOnly:           true,
			},
			"fingerprint": schema.StringAttribute{
				MarkdownDescription: "Fingerprint of the secret value. Change this to trigger rotation of the associated write-only attribute `value`. " +
					"Can be omitted if resource is imported, in this case the hash is used as an initial fingerprint (computed output).",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"hash": schema.StringAttribute{
				MarkdownDescription: "Hash value of the secret stored in the backend. " +
					"If this hash has changed without changes in the version attribute, the secret was changed externally.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		}
	}

	// secretAttributeTypes is also used in this package to configure the underlying basetypes.ObjectType.
	secretAttributeTypes = maps.MapValues(secretAttributes(AttributeSchemaOptions{}), func(from schema.Attribute) attr.Type {
		return from.GetType()
	})
)
