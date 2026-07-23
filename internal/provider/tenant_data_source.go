package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &tenantDataSource{}
	_ datasource.DataSourceWithConfigure = &tenantDataSource{}
)

func NewTenantDataSource() datasource.DataSource {
	return &tenantDataSource{}
}

type tenantDataSource struct {
	meshTenantClient client.MeshTenantClient
}

func (d *tenantDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tenant"
}

func (d *tenantDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Single tenant by workspace, project, and platform.",

		Attributes: map[string]schema.Attribute{
			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Tenant metadata. Workspace, project and platform of the target tenant must be set here.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"owned_by_workspace":  schema.StringAttribute{Required: true, MarkdownDescription: "Identifier of the workspace the tenant belongs to."},
					"owned_by_project":    schema.StringAttribute{Required: true, MarkdownDescription: "Identifier of the project the tenant belongs to."},
					"platform_identifier": schema.StringAttribute{Required: true, MarkdownDescription: "Identifier of the target platform (`<platform>.<location>`)."},
					"uuid":                schema.StringAttribute{Computed: true, MarkdownDescription: "The unique identifier (UUID) of the tenant."},
				},
			},

			"spec": schema.SingleNestedAttribute{
				MarkdownDescription: "Tenant specification.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
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
					"quotas": schema.SetNestedAttribute{
						Computed: true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"key":   schema.StringAttribute{Computed: true},
								"value": schema.Int64Attribute{Computed: true},
							},
						},
					},
				},
			},

			"status": schema.SingleNestedAttribute{
				MarkdownDescription: "Tenant status.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
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
				},
			},
		},
	}
}

func (d *tenantDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.meshTenantClient = client.Tenant
	})...)
}

// tenantDataSourceModel reuses the DTO spec/status directly. Only metadata is bespoke: the singular
// data source keys on the composite workspace/project/platform_identifier the DTO metadata lacks.
type tenantDataSourceModel struct {
	Metadata tenantDataSourceMetadata `tfsdk:"metadata"`
	Spec     client.MeshTenantSpec    `tfsdk:"spec"`
	Status   client.MeshTenantStatus  `tfsdk:"status"`
}

type tenantDataSourceMetadata struct {
	OwnedByWorkspace   string `tfsdk:"owned_by_workspace"`
	OwnedByProject     string `tfsdk:"owned_by_project"`
	PlatformIdentifier string `tfsdk:"platform_identifier"`
	Uuid               string `tfsdk:"uuid"`
}

func (d *tenantDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var workspace, project, platform string
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("owned_by_workspace"), &workspace)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("owned_by_project"), &project)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("platform_identifier"), &platform)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// The v4 GA singular GET is by uuid, which composite-key callers don't have; resolve via the list
	// endpoint filtered by workspace/project/platform.
	tenants, err := d.meshTenantClient.List(ctx, client.MeshTenantQuery{Workspace: workspace, Project: &project, Platform: &platform})
	if err != nil {
		resp.Diagnostics.AddError("Unable to read tenant", err.Error())
		return
	}

	// The backend list returns only active tenants by default (soft-deleted and marked-for-deletion
	// are excluded), so no client-side lifecycle filter is needed.
	if len(tenants) != 1 {
		resp.Diagnostics.AddError(
			"Tenant not found",
			fmt.Sprintf("Expected exactly one active tenant for workspace '%s', project '%s', platform '%s', found %d.", workspace, project, platform, len(tenants)),
		)
		return
	}
	tenant := tenants[0]

	model := tenantDataSourceModel{
		Metadata: tenantDataSourceMetadata{
			OwnedByWorkspace:   tenant.Metadata.OwnedByWorkspace,
			OwnedByProject:     tenant.Metadata.OwnedByProject,
			PlatformIdentifier: platform,
			Uuid:               tenant.Metadata.Uuid,
		},
		Spec:   tenant.Spec,
		Status: tenant.Status,
	}

	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, model, tenantConverterOptions()...)...)
}
