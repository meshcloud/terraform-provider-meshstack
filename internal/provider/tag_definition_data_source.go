package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/meshcloud/terraform-provider-meshstack/client"
)

func NewTagDefinitionDataSource() datasource.DataSource {
	return &tagDefinitionDataSource{}
}

type tagDefinitionDataSource struct {
	client *client.MeshStackProviderClient
}

func (d *tagDefinitionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag_definition"
}

func (d *tagDefinitionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A single tag definition by name.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the tag definition.",
				Required:            true,
			},
			"tag_definition": schema.SingleNestedAttribute{
				MarkdownDescription: "Tag definition details",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"kind": schema.StringAttribute{
						MarkdownDescription: "As a common meshObject structure exists, every meshObject has a kind. This is always meshTagDefinition for this endpoint.",
						Computed:            true,
					},
					"api_version": schema.StringAttribute{
						MarkdownDescription: "API Version of meshTagDefinition datatype. Matches the version part provided within the Accept request header.",
						Computed:            true,
					},
					"metadata": schema.SingleNestedAttribute{
						MarkdownDescription: "Always contains the 'name' to uniquely identify the meshTagDefinition.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"name": schema.StringAttribute{
								MarkdownDescription: "Must be of the form $targetKind.$key since tag definitions must be non-conflicting.",
								Computed:            true,
							},
						},
					},
					"spec": schema.SingleNestedAttribute{
						MarkdownDescription: "Specification for the meshTagDefinition.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"target_kind": schema.StringAttribute{
								MarkdownDescription: "The kind of meshObject this tag is defined for. Must be one of: meshProject, meshWorkspace, meshLandingZone, meshPaymentMethod or meshBuildingBlockDefinition.",
								Computed:            true,
							},
							"key": schema.StringAttribute{
								MarkdownDescription: "The key of the tag.",
								Computed:            true,
							},
							"value_type": schema.SingleNestedAttribute{
								MarkdownDescription: "The TagValueType of the tag. Must define exactly one of the available types.",
								Computed:            true,
								Attributes: map[string]schema.Attribute{
									"string": schema.SingleNestedAttribute{
										MarkdownDescription: "string, represented as JSON string",
										Computed:            true,
										Attributes: map[string]schema.Attribute{
											"default_value": schema.StringAttribute{
												MarkdownDescription: "The default value of the tag.",
												Computed:            true,
											},
											"validation_regex": schema.StringAttribute{
												MarkdownDescription: "The regex pattern that the tag value must match.",
												Computed:            true,
											},
										},
									},
									"email": schema.SingleNestedAttribute{
										MarkdownDescription: "email address, represented as JSON string",
										Computed:            true,
										Attributes: map[string]schema.Attribute{
											"default_value": schema.StringAttribute{
												MarkdownDescription: "The default value of the tag.",
												Computed:            true,
											},
											"validation_regex": schema.StringAttribute{
												MarkdownDescription: "The regex pattern that the tag value must match.",
												Computed:            true,
											},
										},
									},
									"integer": schema.SingleNestedAttribute{
										MarkdownDescription: "an integer, represented as a JSON number",
										Computed:            true,
										Attributes: map[string]schema.Attribute{
											"default_value": schema.Int64Attribute{
												MarkdownDescription: "The default value of the tag.",
												Computed:            true,
											},
										},
									},
									"number": schema.SingleNestedAttribute{
										MarkdownDescription: "a decimal number, represented as a JSON number",
										Computed:            true,
										Attributes: map[string]schema.Attribute{
											"default_value": schema.Float64Attribute{
												MarkdownDescription: "The default value of the tag.",
												Computed:            true,
											},
										},
									},
									"single_select": schema.SingleNestedAttribute{
										MarkdownDescription: "a string from a list of options, represented as a JSON string",
										Computed:            true,
										Attributes: map[string]schema.Attribute{
											"options": schema.ListAttribute{
												MarkdownDescription: "The allowed options for the tag as a string[]",
												Computed:            true,
												ElementType:         types.StringType,
											},
											"default_value": schema.StringAttribute{
												MarkdownDescription: "The default value of the tag.",
												Computed:            true,
											},
										},
									},
									"multi_select": schema.SingleNestedAttribute{
										MarkdownDescription: "one or multiple strings from a list of options, represented as a JSON array",
										Computed:            true,
										Attributes: map[string]schema.Attribute{
											"options": schema.ListAttribute{
												MarkdownDescription: "The allowed options for the tag as a string[]",
												Computed:            true,
												ElementType:         types.StringType,
											},
											"default_value": schema.ListAttribute{
												MarkdownDescription: "The default value of the tag.",
												Computed:            true,
												ElementType:         types.StringType,
											},
										},
									},
								},
							},
							"description": schema.StringAttribute{
								MarkdownDescription: "The detailed description of the tag.",
								Computed:            true,
							},
							"display_name": schema.StringAttribute{
								MarkdownDescription: "The display name of the tag.",
								Computed:            true,
							},
							"sort_order": schema.Int64Attribute{
								MarkdownDescription: "The sort order for this tag when displayed in the UI. meshPanel sorts tags in ascending order.",
								Computed:            true,
							},
							"mandatory": schema.BoolAttribute{
								MarkdownDescription: "Indicates whether the tag is mandatory.",
								Computed:            true,
							},
							"immutable": schema.BoolAttribute{
								MarkdownDescription: "Indicates whether the tag value is not editable after initially set.",
								Computed:            true,
							},
							"restricted": schema.BoolAttribute{
								MarkdownDescription: "Indicates whether only admins can edit this tag.",
								Computed:            true,
							},
						},
					},
				},
			},
		},
	}
}

func (d *tagDefinitionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.MeshStackProviderClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *MeshStackProviderClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *tagDefinitionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var name string
	diags := req.Config.GetAttribute(ctx, path.Root("name"), &name)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tag, err := d.client.ReadTagDefinition(name)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read meshTagDefinition", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("tag_definition"), &tag)...)
}
