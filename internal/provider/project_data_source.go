package provider

import (
	"context"
	"fmt"

	"github.com/meshcloud/terraform-provider-meshstack/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &projectDataSource{}
	_ datasource.DataSourceWithConfigure = &projectDataSource{}
)

func NewProjectDataSource() datasource.DataSource {
	return &projectDataSource{}
}

type projectDataSource struct {
	client client.MeshStackProviderClient
}

func (d *projectDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (d *projectDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Single project by name and workspace.",

		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Project datatype version",
				Computed:            true,
			},

			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshProject`.",
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"meshProject"}...),
				},
			},

			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Project metadata. Name and workspace of the target Project must be set here.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"name":               schema.StringAttribute{Required: true},
					"owned_by_workspace": schema.StringAttribute{Required: true},
					"created_on":         schema.StringAttribute{Computed: true},
					"deleted_on":         schema.StringAttribute{Computed: true},
				},
			},

			"spec": schema.SingleNestedAttribute{
				MarkdownDescription: "Project specification.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{

					"display_name": schema.StringAttribute{Computed: true},
					"tags": schema.MapAttribute{
						ElementType: types.ListType{ElemType: types.StringType},
						Computed:    true,
					},
					"payment_method_identifier":            schema.StringAttribute{Computed: true},
					"substitute_payment_method_identifier": schema.StringAttribute{Computed: true},
				},
			},
		},
	}
}

func (d *projectDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(client.MeshStackProviderClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *MeshStackProviderClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *projectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// get workspace and name to query for project
	var workspace, name string
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("owned_by_workspace"), &workspace)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)

	if resp.Diagnostics.HasError() {
		return
	}

	project, err := d.client.Project.Read(workspace, name)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read project", err.Error())
		return
	}

	if project == nil {
		resp.Diagnostics.AddError("Project not found", fmt.Sprintf("Can't find project with identifier '%s' in workspace '%s'.", name, workspace))
		return
	}

	// client data maps directly to the schema so we just need to set the state
	resp.Diagnostics.Append(resp.State.Set(ctx, project)...)
}
