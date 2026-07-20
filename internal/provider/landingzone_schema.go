package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

// landingZoneMetadataDataSourceSchema builds the landing zone metadata block. When computed is false
// (the singular by-name lookup) the block and its `name` are Required inputs; when true (a list
// element) everything is Computed.
func landingZoneMetadataDataSourceSchema(computed bool) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Computed: computed,
		Required: !computed,
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Landing Zone identifier.",
				Computed:            computed,
				Required:            !computed,
			},
			"owned_by_workspace": schema.StringAttribute{
				MarkdownDescription: "Identifier of the workspace that owns this landing zone.",
				Computed:            true,
			},
			"tags": tagsAttribute(tagsOptions{Kind: client.MeshObjectKind.LandingZone, Output: true}),
		},
	}
}

func landingZoneSpecDataSourceSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		Computed: true,
		Attributes: map[string]schema.Attribute{
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Display name of the landing zone.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the landing zone.",
				Computed:            true,
			},
			"automate_deletion_approval": schema.BoolAttribute{
				MarkdownDescription: "Whether deletion approval is automated for this landing zone.",
				Computed:            true,
			},
			"automate_deletion_replication": schema.BoolAttribute{
				MarkdownDescription: "Whether deletion replication is automated for this landing zone.",
				Computed:            true,
			},
			"info_link": schema.StringAttribute{
				MarkdownDescription: "Link to additional information about the landing zone.",
				Computed:            true,
			},
			"platform_ref": meshRefByUuid(meshRefOptions{Kind: client.MeshObjectKind.Platform, Description: "Reference to the platform this landing zone belongs to.", Output: true}),
			"platform_properties": schema.SingleNestedAttribute{
				MarkdownDescription: "Platform-specific configuration options.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"aws":        awsPlatformConfigSchema(),
					"aks":        aksPlatformConfigSchema(),
					"azure":      azurePlatformConfigSchema(),
					"azurerg":    azureRgPlatformConfigSchema(),
					"custom":     customPlatformConfigSchema(),
					"gcp":        gcpPlatformConfigSchema(),
					"kubernetes": kubernetesPlatformConfigSchema(),
					"openshift":  openShiftPlatformConfigSchema(),
					"type": schema.StringAttribute{
						MarkdownDescription: "Type of the platform. This field is automatically inferred from which platform configuration is provided and cannot be set manually.",
						Computed:            true,
					},
				},
			},
			"quotas": schema.ListNestedAttribute{
				MarkdownDescription: "Quota definitions for this landing zone.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							MarkdownDescription: "Quota key identifier.",
							Computed:            true,
						},
						"value": schema.Int64Attribute{
							MarkdownDescription: "Quota value.",
							Computed:            true,
						},
					},
				},
			},
			"mandatory_building_block_refs": schema.ListNestedAttribute{
				MarkdownDescription: "List of mandatory building block references for this landing zone.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"kind": schema.StringAttribute{
							MarkdownDescription: "meshObject type, always `meshBuildingBlockDefinition`.",
							Computed:            true,
						},
						"uuid": schema.StringAttribute{
							MarkdownDescription: "UUID of the building block.",
							Computed:            true,
						},
					},
				},
			},
			"recommended_building_block_refs": schema.ListNestedAttribute{
				MarkdownDescription: "List of recommended building block references for this landing zone.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"kind": schema.StringAttribute{
							MarkdownDescription: "meshObject type, always `meshBuildingBlockDefinition`.",
							Computed:            true,
						},
						"uuid": schema.StringAttribute{
							MarkdownDescription: "UUID of the building block.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func landingZoneStatusDataSourceSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Current Landing Zone status.",
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			"disabled": schema.BoolAttribute{
				MarkdownDescription: "True if the landing zone is disabled.",
				Computed:            true,
			},
			"restricted": schema.BoolAttribute{
				MarkdownDescription: "If true, users will be unable to select this landing zone in meshPanel. " +
					"Only Platform teams can create tenants using restricted landing zones with the meshObject API.",
				Computed: true,
			},
		},
	}
}
