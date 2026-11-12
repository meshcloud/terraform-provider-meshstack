package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/meshcloud/meshstack-cli/client"
)

func tenantSpecDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"platform_ref": meshRefByUuid(meshRefOptions{
			Kind:        client.MeshObjectKind.Platform,
			Description: "Reference to the platform this tenant belongs to, identified by its uuid.",
			Output:      true,
		}),
		"platform_tenant_id": schema.StringAttribute{Computed: true},
		"landing_zone_ref": meshRefByName(meshRefOptions{
			Kind:        client.MeshObjectKind.LandingZone,
			Description: "Reference to the landing zone assigned to this tenant, identified by its name (the landing zone identifier).",
			Output:      true,
		}),
		"requested_quotas": schema.MapNestedAttribute{
			MarkdownDescription: "The quotas requested for this tenant at creation, as a map keyed by quota key " +
				"whose value is an object carrying the requested `value`. This is a create-time input that the " +
				"meshStack API does not return on read, so it is typically null here; read the effective quotas " +
				"from `status.applied_quotas` instead.",
			Computed: true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"value": schema.Int64Attribute{Computed: true},
				},
			},
		},
	}
}

func tenantStatusDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"tenant_name": schema.StringAttribute{
			MarkdownDescription: "Name of the tenant, currently the owning workspace, project and platform (instance) identifiers joined by dots (`<workspace>.<project>.<platform>.<location>`). Treat this as an opaque string and do not parse it: the format is not guaranteed and may change unexpectedly, for example when the location segment becomes optional or when a tenant is moved across projects.",
			Computed:            true,
		},
		"platform_type_identifier": schema.StringAttribute{
			MarkdownDescription: "Identifier of the tenant's platform type — the kind of platform (e.g. `aws`, `azure`), not the specific platform instance the tenant lives on.",
			Computed:            true,
		},
		"platform_workspace_id": schema.StringAttribute{
			MarkdownDescription: "For platforms that represent a workspace as a platform-side container (e.g. a Cloud Foundry Organization or an OpenStack Domain), the platform's own id of that container (an id assigned by the external platform, not a meshWorkspace identifier). Null for platforms with no such concept or until the tenant has been replicated.",
			Computed:            true,
		},
		"tags": schema.MapAttribute{
			MarkdownDescription: "Tags assigned to this tenant.",
			ElementType:         types.ListType{ElemType: types.StringType},
			Computed:            true,
		},
		"applied_quotas": schema.MapNestedAttribute{
			MarkdownDescription: "The effective quotas meshStack applied to this tenant, as a map keyed by quota key whose value is an object carrying the applied `value`. These are the quotas in effect: the landing zone's default quotas, overlaid with the quotas requested at creation and with any later change by a platform operator or an approved quota request.",
			Computed:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"value": schema.Int64Attribute{Computed: true},
				},
			},
		},
	}
}
