package provider

import (
	"context"
	"fmt"

	"github.com/meshcloud/terraform-provider-meshstack/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &projectsDataSource{}
	_ datasource.DataSourceWithConfigure = &projectsDataSource{}
)

func NewProjectsDataSource() datasource.DataSource {
	return &projectsDataSource{}
}

type projectsDataSource struct {
	client *client.MeshStackProviderClient
}

func (d *projectsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_projects"
}

func (d *projectsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Projects in a workspace.",

		Attributes: map[string]schema.Attribute{
			"workspace_identifier": schema.StringAttribute{
				MarkdownDescription: "Workspace identifier",
				Required:            true,
			},
			"payment_method_identifier": schema.StringAttribute{
				MarkdownDescription: "Payment method identifier",
				Optional:            true,
			},
			"projects": schema.ListNestedAttribute{
				MarkdownDescription: "List of projects",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"api_version": schema.StringAttribute{
							MarkdownDescription: "API version",
							Computed:            true,
						},
						"kind": schema.StringAttribute{
							MarkdownDescription: "Kind of project",
							Computed:            true,
						},
						"metadata": schema.SingleNestedAttribute{
							MarkdownDescription: "Metadata of the project",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{
									MarkdownDescription: "Name of the project",
									Computed:            true,
								},
								"owned_by_workspace": schema.StringAttribute{
									MarkdownDescription: "Workspace that owns the project",
									Computed:            true,
								},
								"created_on": schema.StringAttribute{
									MarkdownDescription: "Creation date of the project",
									Computed:            true,
								},
								"deleted_on": schema.StringAttribute{
									MarkdownDescription: "Deletion date of the project",
									Computed:            true,
								},
							},
						},
						"spec": schema.SingleNestedAttribute{
							MarkdownDescription: "Specifications of the project",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"display_name": schema.StringAttribute{
									MarkdownDescription: "Display name of the project",
									Computed:            true,
								},
								"tags": schema.MapAttribute{
									MarkdownDescription: "Tags associated with the project",
									Computed:            true,
									ElementType:         types.ListType{ElemType: types.StringType},
								},
								"payment_method_identifier": schema.StringAttribute{
									MarkdownDescription: "Payment method identifier",
									Computed:            true,
								},
								"substitute_payment_method_identifier": schema.StringAttribute{
									MarkdownDescription: "Substitute payment method identifier",
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

func (d *projectsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.MeshStackProviderClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *MeshStackProviderClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *projectsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var workspaceIdentifier string
	var paymentMethodIdentifier *string

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("workspace_identifier"), &workspaceIdentifier)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("payment_method_identifier"), &paymentMethodIdentifier)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projects, err := d.client.ReadProjects(workspaceIdentifier, paymentMethodIdentifier)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read projects", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("projects"), &projects)...)
}
