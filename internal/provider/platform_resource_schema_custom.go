package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

func customPlatformSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Custom platform configuration.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"platform_type_ref": meshRefByName(meshRefOptions{Kind: client.MeshObjectKind.PlatformType, Description: "Reference to the platform type."}),
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
