package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/meshcloud/meshstack-cli/client"
)

func NewProjectUserBindingDataSource() datasource.DataSource {
	return newProjectBindingDataSource("user", "Username.",
		func(c client.Client) projectBindingClient[client.MeshProjectUserBinding] {
			return c.ProjectUserBinding
		})
}
