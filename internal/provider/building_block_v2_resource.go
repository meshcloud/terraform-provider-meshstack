package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"
	"github.com/meshcloud/terraform-provider-meshstack/client/types/enum"
	"github.com/meshcloud/terraform-provider-meshstack/internal/util/poll"
)

var (
	_ resource.Resource              = &buildingBlockV2Resource{}
	_ resource.ResourceWithConfigure = &buildingBlockV2Resource{}
)

// buildingBlockV2UserInputModel extends buildingBlockUserInputModel with sensitive input variants
// specific to the BB v2 resource. Embedding flattens tfsdk tags so the framework sees all fields.
type buildingBlockV2UserInputModel struct {
	buildingBlockUserInputModel
	ValueStringSensitive types.String `tfsdk:"value_string_sensitive"`
	ValueCodeSensitive   types.String `tfsdk:"value_code_sensitive"`
}

// buildingBlockV2UserInputs extends the base user-inputs schema with sensitive STRING and CODE
// variants. Only used by the BB v2 resource; the v1 buildingblock resource uses the base schema.
func buildingBlockV2UserInputs() schema.MapNestedAttribute {
	inputs := buildingBlockUserInputs()

	// Replace Default with the extended object type (adds the two sensitive fields).
	inputs.Default = mapdefault.StaticValue(
		types.MapValueMust(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"value_string":           types.StringType,
					"value_single_select":    types.StringType,
					"value_multi_select":     types.ListType{ElemType: types.StringType},
					"value_int":              types.Int64Type,
					"value_bool":             types.BoolType,
					"value_code":             types.StringType,
					"value_string_sensitive": types.StringType,
					"value_code_sensitive":   types.StringType,
				},
			},
			map[string]attr.Value{},
		),
	)
	inputs.PlanModifiers = []planmodifier.Map{mapplanmodifier.RequiresReplace()}

	// Extend the ExactlyOneOf validator on value_string to include the sensitive variants.
	inputs.NestedObject.Attributes["value_string"] = schema.StringAttribute{
		Optional: true,
		Computed: false,
		Validators: []validator.String{stringvalidator.ExactlyOneOf(
			path.MatchRelative().AtParent().AtName("value_string"),
			path.MatchRelative().AtParent().AtName("value_single_select"),
			path.MatchRelative().AtParent().AtName("value_multi_select"),
			path.MatchRelative().AtParent().AtName("value_int"),
			path.MatchRelative().AtParent().AtName("value_bool"),
			path.MatchRelative().AtParent().AtName("value_code"),
			path.MatchRelative().AtParent().AtName("value_string_sensitive"),
			path.MatchRelative().AtParent().AtName("value_code_sensitive"),
		)},
	}

	inputs.NestedObject.Attributes["value_string_sensitive"] = schema.StringAttribute{
		MarkdownDescription: "Plaintext value for a sensitive STRING user input. Stored in state but masked in output. " +
			"Use this instead of `value_string` when the building block definition marks the input as sensitive.",
		Optional:  true,
		Sensitive: true,
	}

	inputs.NestedObject.Attributes["value_code_sensitive"] = schema.StringAttribute{
		MarkdownDescription: "Plaintext value for a sensitive CODE user input. Stored in state but masked in output. " +
			"Use this instead of `value_code` when the building block definition marks the input as sensitive.",
		Optional:  true,
		Sensitive: true,
	}

	return inputs
}

func NewBuildingBlockV2Resource() resource.Resource {
	return &buildingBlockV2Resource{}
}

type buildingBlockV2Resource struct {
	meshBuildingBlockV2Client client.MeshBuildingBlockV2Client
}

func (r *buildingBlockV2Resource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_building_block_v2"
}

func (r *buildingBlockV2Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		r.meshBuildingBlockV2Client = client.BuildingBlockV2
	})...)
}

func (r *buildingBlockV2Resource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage a workspace or tenant building block." + previewDisclaimer(),
		DeprecationMessage:  "Use `meshstack_building_block` instead. You can migrate state with a `moved` block from `meshstack_building_block_v2` to `meshstack_building_block`.",

		Attributes: map[string]schema.Attribute{
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
						MarkdownDescription: "References the building block definition version this building block is based on. " +
							"Use `version_latest` or `version_latest_release` from `meshstack_building_block_definition`, or " +
							"`one(data.meshstack_building_block_definitions.<name>.building_block_definitions).version_latest`.",
						Required: true,
						Attributes: map[string]schema.Attribute{
							"uuid": schema.StringAttribute{
								MarkdownDescription: "UUID of the building block definition version.",
								Required:            true,
							},
							"kind": schema.StringAttribute{
								MarkdownDescription: "meshObject type, always `" + client.MeshObjectKind.BuildingBlockDefinitionVersion + "`.",
								Optional:            true,
								Computed:            true,
								Default:             stringdefault.StaticString(client.MeshObjectKind.BuildingBlockDefinitionVersion),
								PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
								Validators:          []validator.String{stringvalidator.OneOf(client.MeshObjectKind.BuildingBlockDefinitionVersion)},
							},
						},
					},

					"target_ref": schema.SingleNestedAttribute{
						MarkdownDescription: "References the building block target. Depending on the definition this must be a workspace or tenant ref. " +
							"For example `data.meshstack_workspace.<name>.ref` or `one(data.meshstack_tenants.<name>.tenants).ref`.",
						Required: true,
						Attributes: map[string]schema.Attribute{
							"kind": schema.StringAttribute{
								MarkdownDescription: "Target kind for this building block, depends on building block definition type. One of `meshTenant`, `meshWorkspace`.",
								Required:            true,
								Validators: []validator.String{
									stringvalidator.OneOf([]string{client.MeshObjectKind.Tenant, client.MeshObjectKind.Workspace}...),
								},
							},
							"uuid": schema.StringAttribute{
								MarkdownDescription: "UUID of the target tenant.",
								Optional:            true,
								Default:             nil,
								Validators: []validator.String{stringvalidator.ExactlyOneOf(
									path.MatchRelative().AtParent().AtName("uuid"),
									path.MatchRelative().AtParent().AtName("name"),
								)},
							},
							"name": schema.StringAttribute{
								MarkdownDescription: "Identifier of the target workspace.",
								Optional:            true,
								Default:             nil,
							},
						},
					},

					"inputs":          buildingBlockV2UserInputs(),
					"combined_inputs": buildingBlockCombinedInputs(),

					"parent_building_blocks": schema.SetNestedAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "Set of parent building blocks.",
						Default: setdefault.StaticValue(
							types.SetValueMust(
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
						MarkdownDescription: "Execution status. One of " + client.BuildingBlockStatuses.Markdown() + ".",
						Computed:            true,
					},
					"force_purge": schema.BoolAttribute{
						MarkdownDescription: "Indicates whether an operator has requested purging of this Building Block.",
						Computed:            true,
					},
					"outputs": buildingBlockOutputs(),
					"lifecycle": schema.SingleNestedAttribute{
						MarkdownDescription: "Lifecycle state of this building block.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"state": schema.StringAttribute{
								MarkdownDescription: "Lifecycle state. One of " + client.BuildingBlockLifecycleStates.Markdown() + ".",
								Computed:            true,
							},
						},
					},
				},
			},
			"wait_for_completion": schema.BoolAttribute{
				MarkdownDescription: "Whether to wait for the Building Block to reach a terminal state (SUCCEEDED or FAILED) before completing the resource creation or deletion. If false, the resource creation completes immediately after the Building Block is created. (Defaults to `true`)",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"purge_on_delete": schema.BoolAttribute{
				MarkdownDescription: "When `true`, deletion skips the Building Block's configured deletion run and immediately removes it from meshStack. Useful when the Building Block is stuck in a non-final state and cannot be deleted normally. Requires `ADM_BUILDINGBLOCK_DELETE` permission. (Defaults to `false`)",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
	}
}

func (r *buildingBlockV2Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	bb := client.MeshBuildingBlockV2{
		Spec: client.MeshBuildingBlockV2Spec{
			BuildingBlockDefinitionVersionRef: client.MeshBuildingBlockV2DefinitionVersionRef{},
			TargetRef:                         client.MeshBuildingBlockV2TargetRef{},
			Inputs:                            make(map[string]*client.MeshBuildingBlockInput),
		},
	}

	// Retrieve values from plan
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("display_name"), &bb.Spec.DisplayName)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("building_block_definition_version_ref").AtName("uuid"), &bb.Spec.BuildingBlockDefinitionVersionRef.Uuid)...)
	bb.Spec.BuildingBlockDefinitionVersionRef.Kind = client.MeshObjectKind.BuildingBlockDefinitionVersion
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("parent_building_blocks"), &bb.Spec.ParentBuildingBlocks)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("target_ref"), &bb.Spec.TargetRef)...)

	// Set user inputs — use the v2-extended model to capture sensitive variants.
	var userInputs map[string]buildingBlockV2UserInputModel
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("inputs"), &userInputs)...)

	for key, values := range userInputs {
		var inputValue clientTypes.SecretOrAny
		var valueType string
		var isSensitive bool

		// Sensitive inputs must be sent as SecretEmbedded {"plaintext": "..."} per the v2-preview API.
		if !values.ValueStringSensitive.IsNull() && !values.ValueStringSensitive.IsUnknown() {
			plaintext := values.ValueStringSensitive.ValueString()
			inputValue = clientTypes.SecretOrAny{X: clientTypes.Secret{Plaintext: &plaintext}}
			valueType = client.MESH_BUILDING_BLOCK_IO_TYPE_STRING
			isSensitive = true
		} else if !values.ValueCodeSensitive.IsNull() && !values.ValueCodeSensitive.IsUnknown() {
			plaintext := values.ValueCodeSensitive.ValueString()
			inputValue = clientTypes.SecretOrAny{X: clientTypes.Secret{Plaintext: &plaintext}}
			valueType = client.MESH_BUILDING_BLOCK_IO_TYPE_CODE
			isSensitive = true
		} else {
			value, vt := values.extractIoValue()
			if value == nil {
				resp.Diagnostics.AddAttributeError(
					path.Root("spec").AtName("inputs"),
					"Input with missing value",
					fmt.Sprintf("Input '%s' must have one value field set.", key),
				)
			}
			inputValue = clientTypes.SecretOrAny{Y: value}
			valueType = vt
		}
		// IsSensitive must be set so the secret stays in the SecretOrAny.X variant across a JSON
		// round-trip (MeshBuildingBlockInput.UnmarshalJSON demotes X→Y when it is false). The v3
		// resource sets it symmetrically (buildingBlockConverterOptions); without it the mock's
		// deep-copy strips the Secret and the plaintext leaks into value_string as a raw map.
		bb.Spec.Inputs[key] = &client.MeshBuildingBlockInput{
			Value:       inputValue,
			ValueType:   new(enum.Entry[client.MeshBuildingBlockIOType](valueType)),
			IsSensitive: isSensitive,
		}
	}

	var waitForCompletion bool
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("wait_for_completion"), &waitForCompletion)...)
	var purgeOnDelete bool
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("purge_on_delete"), &purgeOnDelete)...)

	// Check errors after reading plan
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.meshBuildingBlockV2Client.Create(ctx, &bb)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating building block",
			"Could not create building block, unexpected error: "+err.Error(),
		)
		return
	}
	resp.Diagnostics.Append(setStateFromResponseV2(ctx, &resp.State, created)...)

	// ensure that user inputs, wait_for_completion, and purge_on_delete are passed along from the plan
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("inputs"), userInputs)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("wait_for_completion"), waitForCompletion)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("purge_on_delete"), purgeOnDelete)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Pollable for completion if wait_for_completion is true
	if waitForCompletion {
		var lastPollOutput *client.MeshBuildingBlockV2
		err := poll.AtMostFor(30*time.Minute, r.meshBuildingBlockV2Client.ReadFunc(*created.Metadata.Uuid), poll.WithLastResultTo(&lastPollOutput)).
			Until(ctx, (*client.MeshBuildingBlockV2).CreateSuccessful)
		if lastPollOutput != nil {
			// Always set last known building block state, no matter what error!
			resp.Diagnostics.Append(setStateFromResponseV2(ctx, &resp.State, lastPollOutput)...)
		}
		if err != nil {
			resp.Diagnostics.AddError("Failed to await building block creation", err.Error())
			return
		}
		if lastPollOutput.IsWaitingForInput() {
			resp.Diagnostics.AddWarning(
				"Building block run is waiting for input",
				fmt.Sprintf("Building block %s is in status %s. Provide the required inputs in meshPanel to complete the run.", *created.Metadata.Uuid, lastPollOutput.Status.Status),
			)
		}
	}
}

func (r *buildingBlockV2Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var uuid string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bb, err := r.meshBuildingBlockV2Client.Read(ctx, uuid)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read building block", err.Error())
	}

	// The block is gone when the read 404s (nil, e.g. after a hard delete/purge) or when the backend
	// returns it soft-deleted (lifecycle DELETED — a soft delete does not 404). Either way, drop it.
	if bb == nil || (bb.Status != nil && bb.Status.Lifecycle.State == client.BuildingBlockLifecycleStateDeleted) {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(setStateFromResponseV2(ctx, &resp.State, bb)...)
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

	// Get the purge_on_delete setting from the current state
	var purgeOnDelete types.Bool
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("purge_on_delete"), &purgeOnDelete)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.meshBuildingBlockV2Client.Delete(ctx, uuid, purgeOnDelete.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting building block",
			"Could not delete building block, unexpected error: "+err.Error(),
		)
		return
	}

	// Always poll for deletion completion so dependent resources (e.g. the BBD) can be cleaned up safely.
	if err := poll.AtMostFor(30*time.Minute, r.meshBuildingBlockV2Client.ReadFunc(uuid)).
		Until(ctx, (*client.MeshBuildingBlockV2).DeletionSuccessful); err != nil {
		resp.Diagnostics.AddError("Failed to await building block deletion", err.Error())
	}
}

// TODO: A clean import requires us to be able to read the building block definition so that we can differentiate between user and operator/static inputs.
// func (r *buildingBlockV2Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
// 	resource.ImportStatePassthroughID(ctx, path.Root("metadata").AtName("uuid"), req, resp)
// }

func toResourceModelV2Input(key string, io client.MeshBuildingBlockInput, diags *diag.Diagnostics) buildingBlockIoModel {
	value := io.Value.Y
	if io.Value.HasX() && io.Value.X.Hash != nil {
		// Sensitive inputs: the API returns only the secret hash (never plaintext).
		// Surface it so the input is represented in state instead of silently dropped.
		value = *io.Value.X.Hash
	}
	var vt string
	if io.ValueType != nil {
		vt = string(*io.ValueType)
	}
	return toResourceModel(client.MeshBuildingBlockIO{Key: key, Value: value, ValueType: vt}, diags)
}

func setStateFromResponseV2(ctx context.Context, state *tfsdk.State, bb *client.MeshBuildingBlockV2) (diags diag.Diagnostics) {
	diags.Append(state.SetAttribute(ctx, path.Root("metadata"), bb.Metadata)...)

	diags.Append(state.SetAttribute(ctx, path.Root("spec").AtName("display_name"), bb.Spec.DisplayName)...)
	diags.Append(state.SetAttribute(ctx, path.Root("spec").AtName("building_block_definition_version_ref").AtName("uuid"), bb.Spec.BuildingBlockDefinitionVersionRef.Uuid)...)
	diags.Append(state.SetAttribute(ctx, path.Root("spec").AtName("building_block_definition_version_ref").AtName("kind"), client.MeshObjectKind.BuildingBlockDefinitionVersion)...)
	diags.Append(state.SetAttribute(ctx, path.Root("spec").AtName("target_ref"), bb.Spec.TargetRef)...)
	diags.Append(state.SetAttribute(ctx, path.Root("spec").AtName("parent_building_blocks"), bb.Spec.ParentBuildingBlocks)...)

	combinedInputs := make(map[string]buildingBlockIoModel)
	for key, input := range bb.Spec.Inputs {
		combinedInputs[key] = toResourceModelV2Input(key, *input, &diags)
	}
	if diags.HasError() {
		return
	}
	diags.Append(state.SetAttribute(ctx, path.Root("spec").AtName("combined_inputs"), combinedInputs)...)

	// Status is a response-only object that the backend always populates on a GET (the pointer exists
	// only so it can be omitted from requests), so no nil check is needed here.
	diags.Append(state.SetAttribute(ctx, path.Root("status").AtName("status"), bb.Status.Status)...)
	diags.Append(state.SetAttribute(ctx, path.Root("status").AtName("force_purge"), bb.Status.ForcePurge)...)
	diags.Append(state.SetAttribute(ctx, path.Root("status").AtName("lifecycle"), bb.Status.Lifecycle)...)

	outputs := make(map[string]buildingBlockOutputModel)
	for key, output := range bb.Status.Outputs {
		outputs[key] = toResourceModel(client.MeshBuildingBlockIO{Key: key, Value: output.Value, ValueType: string(output.ValueType)}, &diags).toOutputModel()
	}
	if diags.HasError() {
		return
	}
	diags.Append(state.SetAttribute(ctx, path.Root("status").AtName("outputs"), outputs)...)

	return diags
}
