package provider

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// meshProjectRoleAttribute returns a schema attribute for meshProject role references.
// This is used across multiple resources (landingzone, platform) to maintain consistency.
func meshProjectRoleAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "the meshProject role",
		Required:            true,
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The identifier of the meshProjectRole",
			},
			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshProjectRole`.",
				Computed:            true,
				Default:             stringdefault.StaticString("meshProjectRole"),
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"meshProjectRole"}...),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}
