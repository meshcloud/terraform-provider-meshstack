package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/meshcloud/meshstack-cli/client"
)

type projectGroupBindingModel struct {
	projectBindingModel
	Status *client.MeshGroupBindingStatus `tfsdk:"status"`
}

func NewProjectGroupBindingDataSource() datasource.DataSource {
	dataSource := newProjectBindingDataSource("group", "Identifier of the workspace user group.",
		func(c client.Client) projectBindingClient[client.MeshProjectGroupBinding] {
			return c.ProjectGroupBinding
		},
		func(binding client.MeshProjectGroupBinding) projectGroupBindingModel {
			return projectGroupBindingModel{newProjectBindingModel(binding.MeshProjectBinding), binding.Status}
		})
	dataSource.attributes["status"] = groupBindingStatusDataSourceAttribute()
	return dataSource
}
