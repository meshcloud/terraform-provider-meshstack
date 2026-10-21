package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/meshcloud/meshstack-cli/client"
)

type workspaceUserBindingModel struct {
	workspaceBindingModel
	Status *client.MeshUserBindingStatus `tfsdk:"status"`
}

func NewWorkspaceUserBindingDataSource() datasource.DataSource {
	dataSource := newWorkspaceBindingDataSource("user", "Username.", true,
		func(c client.Client) workspaceBindingClient[client.MeshWorkspaceUserBinding] {
			return c.WorkspaceUserBinding
		},
		func(binding client.MeshWorkspaceUserBinding) workspaceUserBindingModel {
			return workspaceUserBindingModel{newWorkspaceBindingModel(binding.MeshWorkspaceBinding), binding.Status}
		})
	dataSource.attributes["status"] = userBindingStatusDataSourceAttribute()
	return dataSource
}
