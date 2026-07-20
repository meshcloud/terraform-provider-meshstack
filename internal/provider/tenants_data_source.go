package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

var (
	_ datasource.DataSource              = &tenantsDataSource{}
	_ datasource.DataSourceWithConfigure = &tenantsDataSource{}
)

func NewTenantsDataSource() datasource.DataSource {
	return &tenantsDataSource{}
}

type tenantsDataSource struct {
	meshTenantClient client.MeshTenantClient
}

func (d *tenantsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tenants"
}

func (d *tenantsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.meshTenantClient = client.Tenant
	})...)
}

func (d *tenantsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Query tenants in a workspace with optional filters.",

		Attributes: map[string]schema.Attribute{
			"workspace": schema.StringAttribute{
				MarkdownDescription: "Workspace identifier. Required.",
				Required:            true,
			},
			"project": schema.StringAttribute{
				MarkdownDescription: "Project identifier.",
				Optional:            true,
			},
			"platform": schema.StringAttribute{
				MarkdownDescription: "Full platform identifier (e.g. `aws.aws-meshstack-dev`).",
				Optional:            true,
			},
			"platform_type": schema.StringAttribute{
				MarkdownDescription: "Platform type identifier (e.g. `AWS`).",
				Optional:            true,
			},
			"landing_zone": schema.StringAttribute{
				MarkdownDescription: "Landing zone identifier.",
				Optional:            true,
			},
			"platform_tenant_id": schema.StringAttribute{
				MarkdownDescription: "Platform-specific tenant ID.",
				Optional:            true,
			},

			"tenants": schema.SetNestedAttribute{
				MarkdownDescription: "Set of matching tenants.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ref": meshRefByUuid(meshRefOptions{Kind: client.MeshObjectKind.Tenant, Description: "Reference to this tenant, can be used as `target_ref` in building block resources.", Output: true}),
						"metadata": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"uuid": schema.StringAttribute{
									Computed: true,
								},
								"owned_by_workspace": schema.StringAttribute{
									Computed: true,
								},
								"owned_by_project": schema.StringAttribute{
									Computed: true,
								},
							},
						},
						"spec": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"platform_ref": meshRefByUuid(meshRefOptions{
									Kind:        client.MeshObjectKind.Platform,
									Description: "Reference to the platform this tenant belongs to, identified by its uuid.",
									Output:      true,
								}),
								"platform_tenant_id": schema.StringAttribute{
									Computed: true,
								},
								"landing_zone_ref": meshRefByName(meshRefOptions{
									Kind:        client.MeshObjectKind.LandingZone,
									Description: "Reference to the landing zone assigned to this tenant, identified by its name (the landing zone identifier).",
									Output:      true,
								}),
								"quotas": schema.ListNestedAttribute{
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
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"tenant_identifier": schema.StringAttribute{
									MarkdownDescription: "Fully-qualified identifier of the tenant: the owning workspace, project and platform (instance) identifiers joined by dots (`<workspace>.<project>.<platform>.<location>`).",
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
				},
			},
		},
	}
}

func (d *tenantsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var workspace string
	var project, platform, platformType, landingZone, platformTenantId *string

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("workspace"), &workspace)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("project"), &project)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("platform"), &platform)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("platform_type"), &platformType)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("landing_zone"), &landingZone)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("platform_tenant_id"), &platformTenantId)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tenants, err := d.meshTenantClient.List(ctx, &client.MeshTenantQuery{
		Workspace:      workspace,
		Project:        project,
		Platform:       platform,
		PlatformType:   platformType,
		LandingZone:    landingZone,
		PlatformTenant: platformTenantId,
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to read tenants", err.Error())
		return
	}

	models := make([]tenantModel, len(tenants))
	for i := range tenants {
		models[i] = tenantModelFromDto(&tenants[i])
	}

	// The `tenants` set and the nested `spec.quotas` list render differently (set vs list); the standard
	// framework honors each per the schema, whereas the generic converter's set detection is global. The
	// mapping here is already model-based (tenantModel), so no hand-rolled attribute plumbing remains.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("tenants"), &models)...)
}
