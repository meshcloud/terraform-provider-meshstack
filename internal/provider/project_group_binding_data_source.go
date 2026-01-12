package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

var (
	_ datasource.DataSource              = &projectGroupBindingDataSource{}
	_ datasource.DataSourceWithConfigure = &projectGroupBindingDataSource{}
)

func NewProjectGroupBindingDataSource() datasource.DataSource {
	return &projectGroupBindingDataSource{}
}

type projectGroupBindingDataSource struct {
	MeshProjectGroupBinding client.MeshProjectGroupBindingClient
}

func (d *projectGroupBindingDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_group_binding"

}

func (d *projectGroupBindingDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Single project group binding by name.",

		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Project group binding datatype version",
				Computed:            true,
			},

			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshProjectGroupBinding`.",
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"meshProjectGroupBinding"}...),
				},
			},

			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Project role assigned by this binding.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required: true,
						Validators: []validator.String{
							stringvalidator.LengthBetween(1, 45),
						},
					},
				},
			},

			"role_ref": schema.SingleNestedAttribute{
				MarkdownDescription: "Project role assigned by this binding.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{Computed: true},
				},
			},

			"target_ref": schema.SingleNestedAttribute{
				MarkdownDescription: "Project, identified by workspace and project identifier.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{

					"name":               schema.StringAttribute{Computed: true},
					"owned_by_workspace": schema.StringAttribute{Computed: true},
				},
			},
			"subject": schema.SingleNestedAttribute{
				MarkdownDescription: "Group assigned by this binding.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Groupname.",
						Computed:            true,
					},
				},
			},
		},
	}
}

func (d *projectGroupBindingDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.MeshProjectGroupBinding = client.ProjectGroupBinding
	})...)
}

func (d *projectGroupBindingDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var name string
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)
	if resp.Diagnostics.HasError() {
		return
	}

	binding, err := d.MeshProjectGroupBinding.Read(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read project group binding", err.Error())
	}

	if binding == nil {
		resp.Diagnostics.AddError("Project group binding not found", fmt.Sprintf("Can't find project group binding with name '%s'.", name))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, binding)...)
}
