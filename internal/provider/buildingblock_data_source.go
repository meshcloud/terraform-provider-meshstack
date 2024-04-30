package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &buildingBlockDataSource{}
	_ datasource.DataSourceWithConfigure = &buildingBlockDataSource{}
)

func NewBuildingBlockDataSource() datasource.DataSource {
	return &buildingBlockDataSource{}
}

type buildingBlockDataSource struct {
	client *MeshStackProviderClient
}

type buildingBlockDataSourceModel IMeshBuildingBlock[types.String, types.Bool, types.Int64]

// Aliases for nested model types
type buildingBlockMetadataModel = IMeshBuildingBlockMetadata[types.String, types.Bool, types.Int64]
type buildingBlockSpecModel = IMeshBuildingBlockSpec[types.String]
type buildingBlockIOModel = IMeshBuildingBlockIO[types.String]
type buildingBlockParentModel = IMeshBuildingBlockParent[types.String]
type buildingBlockStatusModel = IMeshBuildingBlockStatus[types.String]

func (d *buildingBlockDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_buildingblock"
}

func (d *buildingBlockDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "BuildingBlock data source",

		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				Computed: true,
			},

			"kind": schema.StringAttribute{
				Computed: true,
			},

			"metadata": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"uuid":               schema.StringAttribute{Required: true},
					"definition_uuid":    schema.StringAttribute{Computed: true},
					"definition_version": schema.Int64Attribute{Computed: true},
					"tenant_identifier":  schema.StringAttribute{Computed: true},
					"force_purge":        schema.BoolAttribute{Computed: true},
					"created_on":         schema.StringAttribute{Computed: true},
					"marked_for_deletion_on": schema.StringAttribute{
						Computed: true,
					},
					"marked_for_deletion_by": schema.StringAttribute{
						Computed: true,
					},
				},
			},

			"spec": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{

					"display_name": schema.StringAttribute{Computed: true},
					"inputs": schema.ListNestedAttribute{
						Computed: true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"key":        schema.StringAttribute{Computed: true},
								"value":      schema.StringAttribute{Computed: true},
								"value_type": schema.StringAttribute{Computed: true},
							},
						},
					},
					"parent_building_blocks": schema.ListNestedAttribute{
						Computed: true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"buildingblock_uuid": schema.StringAttribute{Computed: true},
								"definition_uuid":    schema.StringAttribute{Computed: true},
							},
						},
					},
				},
			},

			"status": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{

					"status": schema.StringAttribute{Computed: true},
					"outputs": schema.ListNestedAttribute{
						Computed: true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"key":        schema.StringAttribute{Computed: true},
								"value":      schema.StringAttribute{Computed: true},
								"value_type": schema.StringAttribute{Computed: true},
							},
						},
					},
				},
			},
		},
	}
}

func (d *buildingBlockDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*MeshStackProviderClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *MeshStackProviderClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *buildingBlockDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// get UUID for BB we want to query from the request
	var uuid string
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)
	bb, err := d.client.ReadBuildingBlock(uuid)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read buildingblock", err.Error())
	}

	// construct attributes
	metadata := buildingBlockMetadataModel{
		Uuid:                types.StringValue(bb.Metadata.Uuid),
		DefinitionUuid:      types.StringValue(bb.Metadata.DefinitionUuid),
		DefinitionVersion:   types.Int64Value(bb.Metadata.DefinitionVersion),
		TenantIdentifier:    types.StringValue(bb.Metadata.TenantIdentifier),
		ForcePurge:          types.BoolValue(bb.Metadata.ForcePurge),
		CreatedOn:           types.StringValue(bb.Metadata.CreatedOn),
		MarkedForDeletionOn: types.StringValue(bb.Metadata.MarkedForDeletionOn),
		MarkedForDeletionBy: types.StringValue(bb.Metadata.MarkedForDeletionBy),
	}

	specInputs := make([]buildingBlockIOModel, len(bb.Spec.Inputs))
	for i, input := range bb.Spec.Inputs {
		specInputs[i] = buildingBlockIOModel{
			Key:       types.StringValue(input.Key),
			Value:     types.StringValue(input.Value),
			ValueType: types.StringValue(input.ValueType),
		}
	}

	specParents := make([]buildingBlockParentModel, len(bb.Spec.ParentBuildingBlocks))
	for i, parent := range bb.Spec.ParentBuildingBlocks {
		specParents[i] = buildingBlockParentModel{
			BuildingBlockUuid: types.StringValue(parent.BuildingBlockUuid),
			DefinitionUuid:    types.StringValue(parent.DefinitionUuid),
		}
	}

	spec := buildingBlockSpecModel{
		DisplayName:          types.StringValue(bb.Spec.DisplayName),
		Inputs:               specInputs,
		ParentBuildingBlocks: specParents,
	}

	statusOutputs := make([]buildingBlockIOModel, len(bb.Status.Outputs))
	for i, output := range bb.Status.Outputs {
		statusOutputs[i] = buildingBlockIOModel{
			Key:       types.StringValue(output.Key),
			Value:     types.StringValue(output.Value),
			ValueType: types.StringValue(output.ValueType),
		}
	}
	status := buildingBlockStatusModel{
		Status:  types.StringValue(bb.Status.Status),
		Outputs: statusOutputs,
	}

	// assemble set full model
	state := buildingBlockDataSourceModel{
		ApiVersion: types.StringValue(bb.ApiVersion),
		Kind:       types.StringValue(bb.Kind),
		Metadata:   metadata,
		Spec:       spec,
		Status:     status,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
