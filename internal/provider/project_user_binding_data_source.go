package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/meshcloud/meshstack-cli/client"
)

type projectUserBindingModel struct {
	projectBindingModel
	Status *client.MeshUserBindingStatus `tfsdk:"status"`
}

func NewProjectUserBindingDataSource() datasource.DataSource {
	dataSource := newProjectBindingDataSource("user", "Username.",
		func(c client.Client) projectBindingClient[client.MeshProjectUserBinding] {
			return c.ProjectUserBinding
		},
		func(binding client.MeshProjectUserBinding) projectUserBindingModel {
			return projectUserBindingModel{newProjectBindingModel(binding.MeshProjectBinding), binding.Status}
		})
	dataSource.attributes["status"] = userBindingStatusDataSourceAttribute()
	return dataSource
}
