package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var (
	_ datasource.DataSource              = &projectUserBindingsDataSource{}
	_ datasource.DataSourceWithConfigure = &projectUserBindingsDataSource{}
)

func NewProjectUserBindingsDataSource() datasource.DataSource {
	return &projectUserBindingsDataSource{}
}

type projectUserBindingsDataSource struct {
	client *MeshStackProviderClient
}

func (d *projectUserBindingsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_user_binding"

}

func (d *projectUserBindingsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Single project by name and workspace.",

		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Project user binding datatype version",
				Computed:            true,
			},

			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshProjectUserBinding`.",
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"meshProjectUserBinding"}...),
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
				MarkdownDescription: "Users assigned by this binding.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Username.",
						Computed:            true,
					},
				},
			},
		},
	}
}

func (d *projectUserBindingsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *projectUserBindingsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// get workspace and project to query for bindings
	var name string
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)
	if resp.Diagnostics.HasError() {
		return
	}

	binding, err := d.client.ReadProjectUserBinding(name)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read project", err.Error())
	}

	if binding == nil {
		resp.Diagnostics.AddError("Project user binding not found", fmt.Sprintf("Can't find project user binding with name '%s'.", name))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, binding)...)
}
