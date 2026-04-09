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
	_ datasource.DataSource              = &tenantsV4DataSource{}
	_ datasource.DataSourceWithConfigure = &tenantsV4DataSource{}
)

func NewTenantsV4DataSource() datasource.DataSource {
	return &tenantsV4DataSource{}
}

type tenantsV4DataSource struct {
	meshTenantV4Client client.MeshTenantV4Client
}

func (d *tenantsV4DataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tenants"
}

func (d *tenantsV4DataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.meshTenantV4Client = client.TenantV4
	})...)
}

func (d *tenantsV4DataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Query tenants in a workspace with optional filters." + previewDisclaimer(),

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
						"ref": schema.SingleNestedAttribute{
							MarkdownDescription: "Reference to this tenant, can be used as `target_ref` in building block resources.",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"kind": schema.StringAttribute{
									Computed: true,
								},
								"uuid": schema.StringAttribute{
									Computed: true,
								},
							},
						},
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
								"created_on": schema.StringAttribute{
									Computed: true,
								},
								"marked_for_deletion_on": schema.StringAttribute{
									Computed: true,
								},
								"deleted_on": schema.StringAttribute{
									Computed: true,
								},
							},
						},
						"spec": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"platform_identifier": schema.StringAttribute{
									Computed: true,
								},
								"platform_tenant_id": schema.StringAttribute{
									Computed: true,
								},
								"landing_zone_identifier": schema.StringAttribute{
									Computed: true,
								},
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
								"tenant_name": schema.StringAttribute{
									Computed: true,
								},
								"platform_type_identifier": schema.StringAttribute{
									Computed: true,
								},
								"platform_workspace_identifier": schema.StringAttribute{
									Computed: true,
								},
								"tags": schema.MapAttribute{
									ElementType: types.ListType{ElemType: types.StringType},
									Computed:    true,
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *tenantsV4DataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
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

	tenants, err := d.meshTenantV4Client.List(ctx, &client.MeshTenantV4Query{
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

	models := make([]tenantV4Model, len(tenants))
	for i := range tenants {
		models[i] = newTenantV4Model(&tenants[i])
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("tenants"), &models)...)
}
