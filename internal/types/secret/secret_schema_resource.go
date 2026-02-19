package secret

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

type ResourceSchemaOptions struct {
	MarkdownDescription string
	Optional            bool
}

// ResourceSchema defines the Secret representation within the Terraform state/plan.
// Use during Create/Update resource actions with generic.ValueTo, generic.ValueFrom conversion  and WithConverterSupport as options.
// For ModifyPlan resource action, use WalkSecretPathsIn with SetHashToUnknownIfVersionChanged.
func ResourceSchema(opts ResourceSchemaOptions) (result schema.SingleNestedAttribute) {
	return schema.SingleNestedAttribute{
		MarkdownDescription: opts.MarkdownDescription,
		Optional:            opts.Optional,
		Required:            !opts.Optional,
		Attributes: map[string]schema.Attribute{
			valueAttributeKey: schema.StringAttribute{
				MarkdownDescription: opts.MarkdownDescription,
				Required:            true,
				Sensitive:           true,
				WriteOnly:           true,
			},
			versionAttributeKey: schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Version of the secret value. Change this to trigger rotation of the associated write-only attribute `%s`. "+
					"Can be omitted if resource is imported, in this case the `%s` attribute is used as an initial value for this attribute (computed output).", valueAttributeKey, hashAttributeKey),
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			hashAttributeKey: schema.StringAttribute{
				MarkdownDescription: "Hash value of the secret stored in the backend. " +
					"If this hash has changed without changes in the version attribute, the secret value was updated externally.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}
