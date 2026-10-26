package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/meshcloud/meshstack-cli/client"
)

func NewProjectGroupBindingDataSource() datasource.DataSource {
	return newProjectBindingDataSource("group", "Identifier of the workspace user group.",
		func(c client.Client) projectBindingClient[client.MeshProjectGroupBinding] {
			return c.ProjectGroupBinding
		})
}
