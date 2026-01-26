package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func platformTypeMetadataSchema(computed bool) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Platform type metadata",
		Computed:            computed,
		Required:            !computed,
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Unique identifier of the platform type.",
				Computed:            computed,
				Required:            !computed,
			},
			"owned_by_workspace": schema.StringAttribute{
				MarkdownDescription: "Identifier of the workspace that owns this platform type.",
				Computed:            computed,
				Required:            !computed,
			},
			"created_on": schema.StringAttribute{
				MarkdownDescription: "Timestamp of when the platform type was created.",
				Computed:            true,
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the platform type.",
				Computed:            true,
			},
		},
	}
}

func platformTypeSpecSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Specifications of the platform type",
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Display name of the meshPlatformType shown in the UI.",
				Computed:            true,
			},
			"category": schema.StringAttribute{
				MarkdownDescription: "Category of the platform type.",
				Computed:            true,
			},
			"default_endpoint": schema.StringAttribute{
				MarkdownDescription: "Default endpoint URL for platforms of this type.",
				Computed:            true,
			},
			"icon": schema.StringAttribute{
				MarkdownDescription: "Icon used to represent the platform type. Base64 encoded data URI (e.g., `data:image/png;base64,...`).",
				Computed:            true,
			},
		},
	}
}

func platformTypeStatusSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Status of the platform type",
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			"lifecycle": schema.SingleNestedAttribute{
				MarkdownDescription: "Lifecycle information of the platform type",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"state": schema.StringAttribute{
						MarkdownDescription: "Lifecycle state of the platform type. Either ACTIVE or DEACTIVATED.",
						Computed:            true,
					},
				},
			},
		},
	}
}
