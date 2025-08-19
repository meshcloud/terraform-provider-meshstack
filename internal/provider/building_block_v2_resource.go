package provider

import (
	"context"
	"fmt"

	"github.com/meshcloud/terraform-provider-meshstack/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = &buildingBlockV2Resource{}
	_ resource.ResourceWithConfigure = &buildingBlockV2Resource{}
)

func NewBuildingBlockV2Resource() resource.Resource {
	return &buildingBlockV2Resource{}
}

type buildingBlockV2Resource struct {
	client *client.MeshStackProviderClient
}

func (r *buildingBlockV2Resource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_building_block_v2"
}

func (r *buildingBlockV2Resource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.MeshStackProviderClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *MeshStackProviderClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *buildingBlockV2Resource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	mkIoMap := func(isUserInput bool) schema.MapNestedAttribute {
		return schema.MapNestedAttribute{
			Optional: isUserInput,
			Computed: true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"value_string": schema.StringAttribute{
						Optional: isUserInput,
						Computed: !isUserInput,
						Validators: []validator.String{stringvalidator.ExactlyOneOf(
							path.MatchRelative().AtParent().AtName("value_string"),
							path.MatchRelative().AtParent().AtName("value_single_select"),
							path.MatchRelative().AtParent().AtName("value_file"),
							path.MatchRelative().AtParent().AtName("value_int"),
							path.MatchRelative().AtParent().AtName("value_bool"),
							path.MatchRelative().AtParent().AtName("value_list"),
							path.MatchRelative().AtParent().AtName("value_code"),
						)},
					},
					"value_single_select": schema.StringAttribute{Optional: isUserInput, Computed: !isUserInput},
					"value_file":          schema.StringAttribute{Optional: isUserInput, Computed: !isUserInput},
					"value_int":           schema.Int64Attribute{Optional: isUserInput, Computed: !isUserInput},
					"value_bool":          schema.BoolAttribute{Optional: isUserInput, Computed: !isUserInput},
					"value_list": schema.StringAttribute{
						MarkdownDescription: "JSON encoded list of objects.",
						Optional:            isUserInput,
						Computed:            !isUserInput,
					},
					"value_code": schema.StringAttribute{
						MarkdownDescription: "Code value.",
						Optional:            isUserInput,
						Computed:            !isUserInput,
					},
				},
			},
		}
	}

	inputs := mkIoMap(true)
	inputs.MarkdownDescription = "Building block user inputs. Each input has exactly one value. Use the value attribute that corresponds to the desired input type, e.g. `value_int` to set an integer input, and leave the remaining attributes empty."
	inputs.PlanModifiers = []planmodifier.Map{mapplanmodifier.RequiresReplace()}
	inputs.Default = mapdefault.StaticValue(
		types.MapValueMust(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"value_string":        types.StringType,
					"value_single_select": types.StringType,
					"value_file":          types.StringType,
					"value_int":           types.Int64Type,
					"value_bool":          types.BoolType,
					"value_list":          types.StringType,
					"value_code":          types.StringType,
				},
			},
			map[string]attr.Value{},
		),
	)

	combinedInputs := mkIoMap(false)
	combinedInputs.MarkdownDescription = "Contains all building block inputs. Each input has exactly one value attribute set according to its' type."
	combinedInputs.PlanModifiers = []planmodifier.Map{mapplanmodifier.UseStateForUnknown()}

	outputs := mkIoMap(false)
	outputs.MarkdownDescription = "Building block outputs. Each output has exactly one value attribute set."
	outputs.PlanModifiers = []planmodifier.Map{mapplanmodifier.UseStateForUnknown()}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage a workspace or tenant building block.\n\n~> **Note:** This resource is in preview. It's incomplete and will change in the near future.",

		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Building block datatype version",
				Computed:            true,
				Default:             stringdefault.StaticString("v2-preview"),
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
				MarkdownDescription: "Building block metadata.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
				Attributes: map[string]schema.Attribute{
					"uuid": schema.StringAttribute{
						MarkdownDescription: "UUID which uniquely identifies the building block.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()}, // update is not supported, so we need to replace.
					},
					"owned_by_workspace": schema.StringAttribute{
						MarkdownDescription: "The workspace containing this building block.",
						Computed:            true,
					},
					"created_on": schema.StringAttribute{
						MarkdownDescription: "Timestamp of building block creation.",
						Computed:            true,
					},
					"marked_for_deletion_on": schema.StringAttribute{
						MarkdownDescription: "For deleted building blocks: timestamp of deletion.",
						Computed:            true,
					},
					"marked_for_deletion_by": schema.StringAttribute{
						MarkdownDescription: "For deleted building blocks: user who requested deletion.",
						Computed:            true,
					},
				},
			},

			"spec": schema.SingleNestedAttribute{
				MarkdownDescription: "Building block specification.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"display_name": schema.StringAttribute{
						MarkdownDescription: "Display name for the building block as shown in meshPanel.",
						Required:            true,
					},

					"building_block_definition_version_ref": schema.SingleNestedAttribute{
						MarkdownDescription: "References the building block definition this building block is based on.",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"uuid": schema.StringAttribute{
								MarkdownDescription: "UUID of the building block definition version.",
								Required:            true,
							},
						},
					},

					"target_ref": schema.SingleNestedAttribute{
						MarkdownDescription: "References the building block target. Depending on the building block definition this will be a workspace or a tenant",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"kind": schema.StringAttribute{
								MarkdownDescription: "Target kind for this building block, depends on building block definition type. One of `meshTenant`, `meshWorkspace`.",
								Required:            true,
								Validators: []validator.String{
									stringvalidator.OneOf([]string{"meshTenant", "meshWorkspace"}...),
								},
							},
							"uuid": schema.StringAttribute{
								MarkdownDescription: "UUID of the target workspace or tenant.",
								Optional:            true,
								Default:             nil,
								Validators: []validator.String{stringvalidator.ExactlyOneOf(
									path.MatchRelative().AtParent().AtName("uuid"),
									path.MatchRelative().AtParent().AtName("identifier"),
								)},
							},
							"identifier": schema.StringAttribute{
								MarkdownDescription: "Identifier of the target workspace.",
								Optional:            true,
								Default:             nil,
							},
						},
					},

					"inputs":          inputs,
					"combined_inputs": combinedInputs,

					"parent_building_blocks": schema.ListNestedAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "List of parent building blocks.",
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
									MarkdownDescription: "UUID of the parent building block.",
									Required:            true,
								},
								"definition_uuid": schema.StringAttribute{
									MarkdownDescription: "UUID of the parent building block definition.",
									Required:            true,
								},
							},
						},
					},
				},
			},

			"status": schema.SingleNestedAttribute{
				MarkdownDescription: "Current building block status.",
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
					"force_purge": schema.BoolAttribute{
						MarkdownDescription: "Indicates whether an operator has requested purging of this Building Block.",
						Computed:            true,
					},
					"outputs": outputs,
				},
			},
			"wait_for_completion": schema.BoolAttribute{
				MarkdownDescription: "Whether to wait for the Building Block to reach a terminal state (SUCCEEDED or FAILED) before completing the resource creation or deletion. If false, the resource creation completes immediately after the Building Block is created. (Defaults to `true`)",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
		},
	}
}

type buildingBlockV2ResourceModel struct {
	ApiVersion types.String `tfsdk:"api_version"`
	Kind       types.String `tfsdk:"kind"`

	Spec struct {
		DisplayName                       types.String                                     `tfsdk:"display_name"`
		BuildingBlockDefinitionVersionRef buildingBlockV2DefinitionVersionRefResourceModel `tfsdk:"building_block_definition_version_ref"`
		TargetRef                         buildingBlockV2targetRefResourceModel            `tfsdk:"target_ref"`
		ParentBuildingBlocks              types.List                                       `tfsdk:"parent_building_blocks"`
		Inputs                            map[string]buildingBlockIoModel                  `tfsdk:"inputs"`
		CombinedInputs                    types.Map                                        `tfsdk:"combined_inputs"`
	} `tfsdk:"spec"`

	// Metadata and Status are unused when creating the resource
	Metadata types.Object `tfsdk:"metadata"`
	Status   types.Object `tfsdk:"status"`

	// additional attributes not part of the API
	WaitForCompletion types.Bool `tfsdk:"wait_for_completion"`
}

type buildingBlockV2DefinitionVersionRefResourceModel struct {
	Uuid types.String `tfsdk:"uuid"`
}

type buildingBlockV2targetRefResourceModel struct {
	Kind       types.String `tfsdk:"kind"`
	Uuid       types.String `tfsdk:"uuid"`
	Identifier types.String `tfsdk:"identifier"`
}

func (r *buildingBlockV2Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan buildingBlockV2ResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	bb := client.MeshBuildingBlockV2Create{
		ApiVersion: plan.ApiVersion.ValueString(),
		Kind:       plan.Kind.ValueString(),

		Spec: client.MeshBuildingBlockV2Spec{
			DisplayName:          plan.Spec.DisplayName.ValueString(),
			ParentBuildingBlocks: make([]client.MeshBuildingBlockParent, 0),
			BuildingBlockDefinitionVersionRef: client.MeshBuildingBlockV2DefinitionVersionRef{
				Uuid: plan.Spec.BuildingBlockDefinitionVersionRef.Uuid.ValueString(),
			},
			TargetRef: client.MeshBuildingBlockV2TargetRef{
				Kind:       plan.Spec.TargetRef.Kind.ValueString(),
				Uuid:       plan.Spec.TargetRef.Uuid.ValueStringPointer(),
				Identifier: plan.Spec.TargetRef.Identifier.ValueStringPointer(),
			},
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

	created, err := r.client.CreateBuildingBlockV2(&bb)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating building block",
			"Could not create building block, unexpected error: "+err.Error(),
		)
		return
	}

	// Poll for completion if wait_for_completion is true
	if !plan.WaitForCompletion.IsNull() && plan.WaitForCompletion.ValueBool() {
		uuid := created.Metadata.Uuid
		polled, err := r.client.PollBuildingBlockV2UntilCompletion(ctx, uuid)
		if err != nil {
			// Always store the initial state, even if the building block was created in a failed state
			// This allows Terraform to track the resource and handle recreates appropriately
			resp.Diagnostics.Append(setStateFromResponseV2(&ctx, &resp.State, created)...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("status").AtName("status"), "FAILED")...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("inputs"), plan.Spec.Inputs)...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("wait_for_completion"), plan.WaitForCompletion)...)

			resp.Diagnostics.AddError(
				"Error waiting for building block completion",
				err.Error(),
			)
			return
		}
		created = polled
	}

	resp.Diagnostics.Append(setStateFromResponseV2(&ctx, &resp.State, created)...)

	// ensure that user inputs and wait_for_completion are passed along from the plan
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("inputs"), plan.Spec.Inputs)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("wait_for_completion"), plan.WaitForCompletion)...)
}

func (r *buildingBlockV2Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var uuid string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve the wait_for_completion value from the current state since it's not returned by the API
	var currentWaitForCompletion types.Bool
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("wait_for_completion"), &currentWaitForCompletion)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bb, err := r.client.ReadBuildingBlockV2(uuid)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read building block", err.Error())
	}

	if bb == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(setStateFromResponseV2(&ctx, &resp.State, bb)...)

	// Restore the wait_for_completion value from the previous state since it's provider configuration, not API data
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("wait_for_completion"), currentWaitForCompletion)...)
}

func (r *buildingBlockV2Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Building blocks can't be updated", "Unsupported operation: building blocks can't be updated.")
}

func (r *buildingBlockV2Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var uuid string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the wait_for_completion setting from the current state
	var waitForCompletion types.Bool
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("wait_for_completion"), &waitForCompletion)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteBuildingBlockV2(uuid)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting building block",
			"Could not delete building block, unexpected error: "+err.Error(),
		)
		return
	}

	// Poll for completion if wait_for_completion is true
	if !waitForCompletion.IsNull() && waitForCompletion.ValueBool() {
		err := r.client.PollBuildingBlockV2UntilDeletion(ctx, uuid)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error waiting for building block deletion completion",
				err.Error(),
			)
			return
		}
	}
}

// TODO: A clean import requires us to be able to read the building block definition so that we can differentiate between user and operator/static inputs.
// func (r *buildingBlockV2Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
// 	resource.ImportStatePassthroughID(ctx, path.Root("metadata").AtName("uuid"), req, resp)
// }

func setStateFromResponseV2(ctx *context.Context, state *tfsdk.State, bb *client.MeshBuildingBlockV2) diag.Diagnostics {
	diags := make(diag.Diagnostics, 0)

	diags.Append(state.SetAttribute(*ctx, path.Root("api_version"), bb.ApiVersion)...)
	diags.Append(state.SetAttribute(*ctx, path.Root("kind"), bb.Kind)...)

	diags.Append(state.SetAttribute(*ctx, path.Root("metadata"), bb.Metadata)...)

	diags.Append(state.SetAttribute(*ctx, path.Root("spec").AtName("display_name"), bb.Spec.DisplayName)...)
	diags.Append(state.SetAttribute(*ctx, path.Root("spec").AtName("building_block_definition_version_ref"), bb.Spec.BuildingBlockDefinitionVersionRef)...)
	diags.Append(state.SetAttribute(*ctx, path.Root("spec").AtName("target_ref"), bb.Spec.TargetRef)...)
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

	outputs := make(map[string]buildingBlockIoModel)
	for _, output := range bb.Status.Outputs {
		value, err := toResourceModel(&output)

		if err != nil {
			diags.AddError("Error processing output", err.Error())
			return diags
		}

		outputs[output.Key] = *value
	}
	diags.Append(state.SetAttribute(*ctx, path.Root("status").AtName("outputs"), outputs)...)

	return diags
}
