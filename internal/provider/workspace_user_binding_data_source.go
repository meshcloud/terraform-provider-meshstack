package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/meshcloud/meshstack-cli/client"
)

func NewWorkspaceUserBindingDataSource() datasource.DataSource {
	return newWorkspaceBindingDataSource("user", "Username.", true,
		func(c client.Client) workspaceBindingClient[client.MeshWorkspaceUserBinding] {
			return c.WorkspaceUserBinding
		})
}
