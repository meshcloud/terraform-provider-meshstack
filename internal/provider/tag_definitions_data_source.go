package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/meshcloud/meshstack-cli/client"

	"github.com/meshcloud/terraform-provider-meshstack/internal/util/iter"
)

func NewTagDefinitionsDataSource() datasource.DataSource {
	return &tagDefinitionsDataSource{}
}

type tagDefinitionsDataSource struct {
	meshTagDefinitionClient client.MeshTagDefinitionClient
}

func (d *tagDefinitionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag_definitions"
}

func (d *tagDefinitionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Tag definitions for the entire organization.",

		Attributes: map[string]schema.Attribute{
			"tag_definitions": schema.ListNestedAttribute{
				MarkdownDescription: "List of tag definitions",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: tagDefinitionDataSourceAttributes(),
				},
			},
		},
	}
}

func (d *tagDefinitionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.meshTagDefinitionClient = client.TagDefinition
	})...)
}

func (d *tagDefinitionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tags, err := iter.Collect(d.meshTagDefinitionClient.List(ctx, client.NoFilter{}))
	if err != nil {
		resp.Diagnostics.AddError("Unable to read meshTagDefinitions", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("tag_definitions"), &tags)...)
}
