package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	client *MeshStackProviderClient
}

func (d *tenantDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tenant"
}

func (d *tenantDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Single tenant by workspace, project, and platform.",

		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Tenant datatype version",
				Computed:            true,
			},

			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshTenant`.",
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"meshTenant"}...),
				},
			},

			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Tenant metadata. Workspace, project and platform of the target tenant must be set here.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"owned_by_workspace":  schema.StringAttribute{Required: true},
					"owned_by_project":    schema.StringAttribute{Required: true},
					"platform_identifier": schema.StringAttribute{Required: true},
					"deleted_on":          schema.StringAttribute{Computed: true},
					"assigned_tags": schema.MapAttribute{
						ElementType: types.ListType{ElemType: types.StringType},
						Computed:    true,
					},
				},
			},

			"spec": schema.SingleNestedAttribute{
				MarkdownDescription: "Tenant specification.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{

					"local_id":                schema.StringAttribute{Computed: true},
					"landing_zone_identifier": schema.StringAttribute{Computed: true},
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
		},
	}
}

func (d *tenantDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*MeshStackProviderClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *MeshStackProviderClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *tenantDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// get workspace, project and platform to query for tenant
	var workspace, project, platform string
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("owned_by_workspace"), &workspace)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("owned_by_project"), &project)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("platform_identifier"), &platform)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tenant, err := d.client.ReadTenant(workspace, project, platform)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read tenant", err.Error())
		return
	}

	// client data maps directly to the schema so we just need to set the state
	resp.Diagnostics.Append(resp.State.Set(ctx, tenant)...)
}
