package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/meshcloud/meshstack-cli/client"
)

type workspaceGroupBindingModel struct {
	workspaceBindingModel
	Status *client.MeshGroupBindingStatus `tfsdk:"status"`
}

func NewWorkspaceGroupBindingDataSource() datasource.DataSource {
	// The API filters group bindings by member username only.
	const listBySubject = false
	dataSource := newWorkspaceBindingDataSource("group", "Identifier of the workspace user group.", listBySubject,
		func(c client.Client) workspaceBindingClient[client.MeshWorkspaceGroupBinding] {
			return c.WorkspaceGroupBinding
		},
		func(binding client.MeshWorkspaceGroupBinding) workspaceGroupBindingModel {
			return workspaceGroupBindingModel{newWorkspaceBindingModel(binding.MeshWorkspaceBinding), binding.Status}
		})
	dataSource.attributes["status"] = groupBindingStatusDataSourceAttribute()
	return dataSource
}
