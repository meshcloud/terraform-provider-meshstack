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

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &workspaceDataSource{}
	_ datasource.DataSourceWithConfigure = &workspaceDataSource{}
)

func NewWorkspaceDataSource() datasource.DataSource {
	return &workspaceDataSource{}
}

type workspaceDataSource struct {
	MeshWorkspace client.MeshWorkspaceClient
}

func (d *workspaceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace"
}

// Schema defines the schema for the data source.
func (d *workspaceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Read a single workspace by identifier.",

		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Workspace API version.",
				Computed:            true,
			},
			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshWorkspace`.",
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"meshWorkspace"}...),
				},
			},

			"metadata": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Workspace identifier.",
						Required:            true,
					},
					"created_on": schema.StringAttribute{
						MarkdownDescription: "Creation date of the workspace.",
						Computed:            true,
					},
					"deleted_on": schema.StringAttribute{
						MarkdownDescription: "Deletion date of the workspace.",
						Computed:            true,
					},
					"tags": schema.MapAttribute{
						MarkdownDescription: "Tags of the workspace.",
						ElementType:         types.ListType{ElemType: types.StringType},
						Computed:            true,
					},
				},
			},

			"spec": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"display_name": schema.StringAttribute{
						MarkdownDescription: "Display name of the workspace.",
						Computed:            true,
					},
					"platform_builder_access_enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether platform builder access is enabled for the workspace.",
						Computed:            true,
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *workspaceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.MeshWorkspace = client.Workspace
	})...)
}

// Read refreshes the Terraform state with the latest data.
func (d *workspaceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var name string

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)

	if resp.Diagnostics.HasError() {
		return
	}

	workspace, err := d.MeshWorkspace.Read(name)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not read workspace '%s'", name),
			err.Error(),
		)
		return
	}

	if workspace == nil {
		resp.Diagnostics.AddError(
			"Workspace not found",
			fmt.Sprintf("The requested workspace '%s' was not found.", name),
		)
		return
	}

	// client data maps directly to the schema so we just need to set the state
	resp.Diagnostics.Append(resp.State.Set(ctx, workspace)...)
}
