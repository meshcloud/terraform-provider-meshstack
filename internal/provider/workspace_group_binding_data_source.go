package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/meshcloud/meshstack-cli/client"
)

func NewWorkspaceGroupBindingDataSource() datasource.DataSource {
	// The API filters group bindings by member username only.
	const listBySubject = false
	return newWorkspaceBindingDataSource("group", "Identifier of the workspace user group.", listBySubject,
		func(c client.Client) workspaceBindingClient[client.MeshWorkspaceGroupBinding] {
			return c.WorkspaceGroupBinding
		},
		func(binding client.MeshWorkspaceGroupBinding) workspaceBindingModel {
			return newWorkspaceBindingModel(binding.MeshWorkspaceBinding)
		})
}
