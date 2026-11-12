package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/meshcloud/meshstack-cli/client"
)

// platformConfigVisibilityNote documents the cross-workspace config redaction (see the marketplace
// list endpoints): a caller that only *consumes* a platform (not owner/contributor/admin) receives it
// without `spec.config`. Modeled on the building_block_definitions / integrations cross-workspace notes.
const platformConfigVisibilityNote = "\n\n" +
	"Requires the `PLATFORMINSTANCE_LIST` (workspace-scoped) or `ADM_PLATFORMINSTANCE_LIST` API-key right. " +
	"The workspace-scoped right also surfaces platforms **published** to the caller's workspace. " +
	"\n\n" +
	"**Cross-Workspace Access**: for a platform the caller only consumes (i.e. it is published to the " +
	"caller's workspace but not owned/contributed by it), `spec.config` is omitted; owner, contributor and " +
	"admin callers still receive it (secrets are always returned hashed)."

// platformMetadataDataSourceSchema builds the platform metadata block. When computed is false (the
// singular by-uuid lookup) the block and its `uuid` are Required inputs; when true (a list element)
// everything is Computed.
func platformMetadataDataSourceSchema(computed bool) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Computed: computed,
		Required: !computed,
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Platform UUID identifier.",
				Computed:            computed,
				Required:            !computed,
			},
			"name":               computedString("Platform identifier."),
			"owned_by_workspace": computedString("The identifier of the workspace that owns this meshPlatform."),
		},
	}
}

func platformIdentifierDataSourceSchema() schema.Attribute {
	return computedString("Full platform identifier (`<platform-name>.<location-name>`), suitable for use as `platform_identifier` in tenant resources.")
}

func platformSpecDataSourceSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		Computed: true,
		Attributes: map[string]schema.Attribute{
			"display_name":       computedString("The human-readable display name of the meshPlatform."),
			"description":        computedString("Description of the meshPlatform."),
			"endpoint":           computedString("The web console URL endpoint of the platform."),
			"support_url":        computedString("URL for platform support documentation."),
			"documentation_url":  computedString("URL for platform documentation."),
			"access_information": computedString("Free-text access information shown to users when accessing tenants on this platform. Supports markdown formatting."),
			"location_ref":       meshRefByName(meshRefOptions{Kind: client.MeshObjectKind.Location, Description: "Reference to the location where this platform is situated.", Output: true}),
			"contributing_workspaces": schema.SetAttribute{
				MarkdownDescription: "A set of workspace identifiers that contribute to this meshPlatform.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"availability": schema.SingleNestedAttribute{
				MarkdownDescription: "Availability configuration for the meshPlatform.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"restriction":       computedString("Access restriction for the platform. Must be one of: PUBLIC, PRIVATE, RESTRICTED."),
					"publication_state": computedString("Publication state of the platform. Must be one of: PUBLISHED, UNPUBLISHED."),
					"restricted_to_workspaces": schema.SetAttribute{
						MarkdownDescription: "If the restriction is set to `RESTRICTED`, you can specify the workspace identifiers this meshPlatform is restricted to.",
						ElementType:         types.StringType,
						Computed:            true,
					},
				},
			},
			"quota_definitions": schema.SetAttribute{
				MarkdownDescription: "Set of quota definitions for the platform.",
				Computed:            true,
				Sensitive:           false,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"quota_key":               types.StringType,
						"label":                   types.StringType,
						"description":             types.StringType,
						"unit":                    types.StringType,
						"min_value":               types.Int64Type,
						"max_value":               types.Int64Type,
						"auto_approval_threshold": types.Int64Type,
					},
				},
			},
			"config": schema.SingleNestedAttribute{
				MarkdownDescription: "Platform-specific configuration options. Omitted (null) for a platform the caller only consumes cross-workspace; see the data source description.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"custom":     customPlatformDataSourceSchema(),
					"aws":        awsPlatformDataSourceSchema(),
					"aks":        aksPlatformDataSourceSchema(),
					"azure":      azurePlatformDataSourceSchema(),
					"azurerg":    azureRgPlatformDataSourceSchema(),
					"gcp":        gcpPlatformDataSourceSchema(),
					"kubernetes": kubernetesPlatformDataSourceSchema(),
					"openshift":  openShiftPlatformDataSourceSchema(),
					"type":       computedString("Type of the platform."),
				},
			},
		},
	}
}
