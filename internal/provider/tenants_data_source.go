package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/meshcloud/meshstack-cli/client"

	"github.com/meshcloud/terraform-provider-meshstack/internal/util/iter"
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
							Computed:   true,
							Attributes: tenantSpecDataSourceAttributes(),
						},
						"status": schema.SingleNestedAttribute{
							Computed:   true,
							Attributes: tenantStatusDataSourceAttributes(),
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

	tenants, err := iter.Collect(d.meshTenantClient.List(ctx, client.MeshTenantFilter{
		Workspace:      workspace,
		Project:        project,
		Platform:       platform,
		PlatformType:   platformType,
		LandingZone:    landingZone,
		PlatformTenant: platformTenantId,
	}))
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
