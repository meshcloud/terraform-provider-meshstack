package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/meshcloud/meshstack-cli/client"
)

var (
	_ datasource.DataSource              = &workspaceGroupBindingDataSource{}
	_ datasource.DataSourceWithConfigure = &workspaceGroupBindingDataSource{}
)

func NewWorkspaceGroupBindingDataSource() datasource.DataSource {
	return &workspaceGroupBindingDataSource{}
}

type workspaceGroupBindingDataSource struct {
	meshWorkspaceGroupBindingClient client.MeshWorkspaceGroupBindingClient
}

func (d *workspaceGroupBindingDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace_group_binding"
}

func (d *workspaceGroupBindingDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = workspaceBindingDataSourceSchema("group", "Identifier of the workspace user group.")
}

func (d *workspaceGroupBindingDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.meshWorkspaceGroupBindingClient = client.WorkspaceGroupBinding
	})...)
}

func (d *workspaceGroupBindingDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var name string
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)
	if resp.Diagnostics.HasError() {
		return
	}

	binding, err := d.meshWorkspaceGroupBindingClient.Read(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read workspace group binding", err.Error())
		return
	}

	if binding == nil {
		resp.Diagnostics.AddError("Workspace group binding not found", fmt.Sprintf("Can't find workspace group binding with name '%s'.", name))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, binding)...)
}
