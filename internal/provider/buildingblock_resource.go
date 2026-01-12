package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

var (
	_ resource.Resource              = &buildingBlockResource{}
	_ resource.ResourceWithConfigure = &buildingBlockResource{}
)

func NewBuildingBlockResource() resource.Resource {
	return &buildingBlockResource{}
}

type buildingBlockResource struct {
	MeshBuildingBlock client.MeshBuildingBlockClient
}

func (r *buildingBlockResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_buildingblock"
}

func (r *buildingBlockResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		r.MeshBuildingBlock = client.BuildingBlock
	})...)
}

func (r *buildingBlockResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage Building Block assignment.",

		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Building block datatype version",
				Computed:            true,
				Default:             stringdefault.StaticString("v1"),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshBuildingBlock`.",
				Computed:            true,
				Default:             stringdefault.StaticString("meshBuildingBlock"),
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"meshBuildingBlock"}...),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Building Block metadata.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"uuid": schema.StringAttribute{
						MarkdownDescription: "UUID which uniquely identifies the Building Block.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"definition_uuid": schema.StringAttribute{
						MarkdownDescription: "UUID of the Building Block Definition this Building Block is based on.",
						Required:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
					"definition_version": schema.Int64Attribute{
						MarkdownDescription: "Version number of the Building Block Definition this Building Block is based on",
						Required:            true,
						PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
					},
					"tenant_identifier": schema.StringAttribute{
						MarkdownDescription: "Full tenant identifier of the tenant this Building Block is assigned to.",
						Required:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
					"force_purge": schema.BoolAttribute{
						MarkdownDescription: "Indicates whether an operator has requested purging of this Building Block.",
						Computed:            true,
						PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
					},
					"created_on": schema.StringAttribute{
						MarkdownDescription: "Timestamp of Building Block creation.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"marked_for_deletion_on": schema.StringAttribute{
						MarkdownDescription: "For deleted Building Blocks: timestamp of deletion.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"marked_for_deletion_by": schema.StringAttribute{
						MarkdownDescription: "For deleted Building Blocks: user who requested deletion.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
				},
			},

			"spec": schema.SingleNestedAttribute{
				MarkdownDescription: "Building Block specification.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"display_name": schema.StringAttribute{
						MarkdownDescription: "Display name for the Building Block as shown in meshPanel.",
						Required:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},

					"inputs":          buildingBlockUserInputs(),
					"combined_inputs": buildingBlockCombinedInputs(),

					"parent_building_blocks": schema.ListNestedAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "List of parent Building Blocks.",
						Default: listdefault.StaticValue(
							types.ListValueMust(
								types.ObjectType{
									AttrTypes: map[string]attr.Type{
										"buildingblock_uuid": types.StringType,
										"definition_uuid":    types.StringType,
									},
								},
								[]attr.Value{},
							),
						),
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"buildingblock_uuid": schema.StringAttribute{
									MarkdownDescription: "UUID of the parent Building Block.",
									Required:            true,
								},
								"definition_uuid": schema.StringAttribute{
									MarkdownDescription: "UUID of the parent Building Block definition.",
									Required:            true,
								},
							},
						},
					},
				},
			},

			"status": schema.SingleNestedAttribute{
				MarkdownDescription: "Current Building Block status.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
				Attributes: map[string]schema.Attribute{
					"status": schema.StringAttribute{
						MarkdownDescription: "Execution status. One of `WAITING_FOR_DEPENDENT_INPUT`, `WAITING_FOR_OPERATOR_INPUT`, `PENDING`, `IN_PROGRESS`, `SUCCEEDED`, `FAILED`.",
						Computed:            true,
						Validators: []validator.String{
							stringvalidator.OneOf([]string{"WAITING_FOR_DEPENDENT_INPUT", "WAITING_FOR_OPERATOR_INPUT", "PENDING", "IN_PROGRESS", "SUCCEEDED", "FAILED"}...),
						},
					},
					"outputs": buildingBlockOutputs(),
				},
			},
		},
	}
}

type buildingBlockResourceModel struct {
	ApiVersion types.String `tfsdk:"api_version"`
	Kind       types.String `tfsdk:"kind"`

	Metadata struct {
		Uuid                types.String `tfsdk:"uuid"`
		DefinitionUuid      types.String `tfsdk:"definition_uuid"`
		DefinitionVersion   types.Int64  `tfsdk:"definition_version"`
		TenantIdentifier    types.String `tfsdk:"tenant_identifier"`
		ForcePurge          types.Bool   `tfsdk:"force_purge"`
		CreatedOn           types.String `tfsdk:"created_on"`
		MarkedForDeletionOn types.String `tfsdk:"marked_for_deletion_on"`
		MarkedForDeletionBy types.String `tfsdk:"marked_for_deletion_by"`
	} `tfsdk:"metadata"`

	Spec struct {
		DisplayName          types.String                           `tfsdk:"display_name"`
		ParentBuildingBlocks types.List                             `tfsdk:"parent_building_blocks"`
		Inputs               map[string]buildingBlockUserInputModel `tfsdk:"inputs"`
		CombinedInputs       types.Map                              `tfsdk:"combined_inputs"`
	} `tfsdk:"spec"`

	Status types.Object `tfsdk:"status"`
}

func (r *buildingBlockResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan buildingBlockResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	bb := client.MeshBuildingBlockCreate{
		ApiVersion: plan.ApiVersion.ValueString(),
		Kind:       plan.Kind.ValueString(),

		Metadata: client.MeshBuildingBlockCreateMetadata{
			DefinitionUuid:    plan.Metadata.DefinitionUuid.ValueString(),
			DefinitionVersion: plan.Metadata.DefinitionVersion.ValueInt64(),
			TenantIdentifier:  plan.Metadata.TenantIdentifier.ValueString(),
		},

		Spec: client.MeshBuildingBlockSpec{
			DisplayName:          plan.Spec.DisplayName.ValueString(),
			ParentBuildingBlocks: make([]client.MeshBuildingBlockParent, 0),
		},
	}

	// add parent building blocks
	plan.Spec.ParentBuildingBlocks.ElementsAs(ctx, &bb.Spec.ParentBuildingBlocks, false)

	// convert inputs
	bb.Spec.Inputs = make([]client.MeshBuildingBlockIO, 0)
	for key, values := range plan.Spec.Inputs {
		value, valueType := values.extractIoValue()
		if value == nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("spec").AtName("inputs"),
				"Input with missing value",
				fmt.Sprintf("Input '%s' must have one value field set.", key),
			)
		}
		input := client.MeshBuildingBlockIO{
			Key:       key,
			Value:     value,
			ValueType: valueType,
		}
		bb.Spec.Inputs = append(bb.Spec.Inputs, input)
	}

	created, err := r.MeshBuildingBlock.Create(ctx, &bb)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating building block",
			"Could not create building block, unexpected error: "+err.Error(),
		)
		return
	}
	resp.Diagnostics.Append(r.setStateFromResponse(&ctx, &resp.State, created)...)

	// ensure that user inputs are passed along
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("inputs"), plan.Spec.Inputs)...)
}

func (r *buildingBlockResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var uuid string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bb, err := r.MeshBuildingBlock.Read(ctx, uuid)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read building block", err.Error())
	}

	if bb == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(r.setStateFromResponse(&ctx, &resp.State, bb)...)
}

func (r *buildingBlockResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Building blocks can't be updated", "Unsupported operation: building blocks can't be updated.")
}

func (r *buildingBlockResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var uuid string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.MeshBuildingBlock.Delete(ctx, uuid)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting building block",
			"Could not delete building block, unexpected error: "+err.Error(),
		)
		return
	}
}

// TODO: A clean import requires us to be able to read the building block definition so that we can differentiate between user and operator/static inputs.
// func (r *buildingBlockResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
// 	resource.ImportStatePassthroughID(ctx, path.Root("metadata").AtName("uuid"), req, resp)
// }

func toResourceModel(io *client.MeshBuildingBlockIO) (*buildingBlockIoModel, error) {
	resourceIo := buildingBlockIoModel{}
	foundValue := false

	if io.Value == nil {
		return &resourceIo, nil
	}

	switch io.ValueType {
	case client.MESH_BUILDING_BLOCK_IO_TYPE_BOOLEAN:
		value, ok := io.Value.(bool)
		if ok {
			resourceIo.ValueBool = types.BoolValue(value)
			foundValue = true
		}

	case client.MESH_BUILDING_BLOCK_IO_TYPE_INTEGER:
		// float because it's an untyped JSON value
		value, ok := io.Value.(float64)
		if ok {
			resourceIo.ValueInt = types.Int64Value(int64(value))
			foundValue = true
		}

	case client.MESH_BUILDING_BLOCK_IO_TYPE_MULTI_SELECT:
		values, ok := io.Value.([]any)
		if ok {
			multiSelect := make([]types.String, len(values))
			for i, v := range values {
				if str, ok := v.(string); ok {
					multiSelect[i] = types.StringValue(str)
				} else {
					return nil, fmt.Errorf("multi-select value at index %d is not a string for input '%s'", i, io.Key)
				}
			}
			resourceIo.ValueMultiSelect = multiSelect
			foundValue = true
		}

	case client.MESH_BUILDING_BLOCK_IO_TYPE_SINGLE_SELECT:
		value, ok := io.Value.(string)
		if ok {
			resourceIo.ValueSingleSelect = types.StringValue(value)
			foundValue = true
		}

	case client.MESH_BUILDING_BLOCK_IO_TYPE_STRING:
		value, ok := io.Value.(string)
		if ok {
			resourceIo.ValueString = types.StringValue(value)
			foundValue = true
		}

	case client.MESH_BUILDING_BLOCK_IO_TYPE_CODE:
		value, ok := io.Value.(string)
		if ok {
			resourceIo.ValueCode = types.StringValue(value)
			foundValue = true
		}

	case client.MESH_BUILDING_BLOCK_IO_TYPE_FILE:
		value, ok := io.Value.(string)
		if ok {
			resourceIo.ValueFile = types.StringValue(value)
			foundValue = true
		}

	case client.MESH_BUILDING_BLOCK_IO_TYPE_LIST:
		value, err := json.Marshal(io.Value)
		if err != nil {
			return nil, err
		}
		resourceIo.ValueList = types.StringValue(string(value))
		foundValue = true

	default:
		return nil, fmt.Errorf("input type '%s' is not supported", io.ValueType)
	}

	if foundValue {
		return &resourceIo, nil
	}

	return nil, fmt.Errorf("input '%s' with value type '%s' does not match actual value", io.Key, io.ValueType)
}

func (r *buildingBlockResource) setStateFromResponse(ctx *context.Context, state *tfsdk.State, bb *client.MeshBuildingBlock) diag.Diagnostics {
	diags := make(diag.Diagnostics, 0)

	diags.Append(state.SetAttribute(*ctx, path.Root("api_version"), bb.ApiVersion)...)
	diags.Append(state.SetAttribute(*ctx, path.Root("kind"), bb.Kind)...)

	diags.Append(state.SetAttribute(*ctx, path.Root("metadata"), bb.Metadata)...)

	diags.Append(state.SetAttribute(*ctx, path.Root("spec").AtName("display_name"), bb.Spec.DisplayName)...)
	diags.Append(state.SetAttribute(*ctx, path.Root("spec").AtName("parent_building_blocks"), bb.Spec.ParentBuildingBlocks)...)

	combinedInputs := make(map[string]buildingBlockIoModel)
	for _, input := range bb.Spec.Inputs {
		value, err := toResourceModel(&input)

		if err != nil {
			diags.AddError("Error processing input", err.Error())
			return diags
		}

		combinedInputs[input.Key] = *value
	}
	diags.Append(state.SetAttribute(*ctx, path.Root("spec").AtName("combined_inputs"), combinedInputs)...)

	diags.Append(state.SetAttribute(*ctx, path.Root("status").AtName("status"), bb.Status.Status)...)

	outputs := make(map[string]buildingBlockOutputModel)
	for _, output := range bb.Status.Outputs {
		value, err := toResourceModel(&output)

		if err != nil {
			diags.AddError("Error processing output", err.Error())
			return diags
		}

		outputs[output.Key] = value.toOutputModel()
	}
	diags.Append(state.SetAttribute(*ctx, path.Root("status").AtName("outputs"), outputs)...)

	return diags
}
