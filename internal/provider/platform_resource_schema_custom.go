package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func customPlatformSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Custom platform configuration.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"platform_type_ref": schema.SingleNestedAttribute{
				MarkdownDescription: "Reference to the platform type.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Name of the platform type.",
						Required:            true,
					},
					"kind": schema.StringAttribute{
						MarkdownDescription: "Kind of the platform type. Always `meshPlatformType`.",
						Computed:            true,
						Default:             stringdefault.StaticString("meshPlatformType"),
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
				},
			},
			"metering": schema.SingleNestedAttribute{
				MarkdownDescription: "Metering configuration.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"processing": meteringProcessingConfigSchema(),
				},
			},
		},
	}
}
