package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/meshcloud/meshstack-cli/client"
)

var (
	_ datasource.DataSource              = &workspaceUserBindingDataSource{}
	_ datasource.DataSourceWithConfigure = &workspaceUserBindingDataSource{}
)

func NewWorkspaceUserBindingDataSource() datasource.DataSource {
	return &workspaceUserBindingDataSource{}
}

type workspaceUserBindingDataSource struct {
	meshWorkspaceUserBindingClient client.MeshWorkspaceUserBindingClient
}

func (d *workspaceUserBindingDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace_user_binding"
}

func (d *workspaceUserBindingDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = workspaceBindingDataSourceSchema("user", "Username.")
}

func (d *workspaceUserBindingDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.meshWorkspaceUserBindingClient = client.WorkspaceUserBinding
	})...)
}

func (d *workspaceUserBindingDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var name string
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)
	if resp.Diagnostics.HasError() {
		return
	}

	binding, err := d.meshWorkspaceUserBindingClient.Read(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read workspace user binding", err.Error())
		return
	}

	if binding == nil {
		resp.Diagnostics.AddError("Workspace user binding not found", fmt.Sprintf("Can't find workspace user binding with name '%s'.", name))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, binding)...)
}
