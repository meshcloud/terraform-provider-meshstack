package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/secret"
	"github.com/meshcloud/terraform-provider-meshstack/internal/util/poll"
)

var (
	_ resource.Resource                = &buildingBlockV3Resource{}
	_ resource.ResourceWithConfigure   = &buildingBlockV3Resource{}
	_ resource.ResourceWithImportState = &buildingBlockV3Resource{}
	_ resource.ResourceWithModifyPlan  = &buildingBlockV3Resource{}
	_ resource.ResourceWithMoveState   = &buildingBlockV3Resource{}
)

func NewBuildingBlockV3Resource() resource.Resource {
	return &buildingBlockV3Resource{}
}

type buildingBlockV3Resource struct {
	meshBuildingBlockV3Client                client.MeshBuildingBlockV3Client
	meshBuildingBlockRunClient               client.MeshBuildingBlockRunClient
	meshBuildingBlockDefinitionVersionClient client.MeshBuildingBlockDefinitionVersionClient
}

func (r *buildingBlockV3Resource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_building_block_v3"
}

func (r *buildingBlockV3Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		r.meshBuildingBlockV3Client = client.BuildingBlockV3
		r.meshBuildingBlockRunClient = client.BuildingBlockRun
		r.meshBuildingBlockDefinitionVersionClient = client.BuildingBlockDefinitionVersion
	})...)
}

func (r *buildingBlockV3Resource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage a workspace or tenant building block using the meshBuildingBlock v2-preview API transport." + previewDisclaimer(),
		Attributes: map[string]schema.Attribute{
			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Building block metadata.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
				Attributes: map[string]schema.Attribute{
					"uuid": schema.StringAttribute{
						MarkdownDescription: "UUID which uniquely identifies the building block.",
						Computed:            true,
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
						PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
					"building_block_definition_version_ref": schema.SingleNestedAttribute{
						MarkdownDescription: "References the building block definition version this building block is based on.",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"uuid": schema.StringAttribute{
								MarkdownDescription: "UUID of the building block definition version.",
								Required:            true,
								PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
							},
						},
					},
					"target_ref": schema.SingleNestedAttribute{
						MarkdownDescription: "References the building block target. Depending on the definition this must be a workspace or tenant ref.",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"kind": schema.StringAttribute{
								MarkdownDescription: "Target kind for this building block, one of `meshTenant`, `meshWorkspace`.",
								Required:            true,
								Validators: []validator.String{
									stringvalidator.OneOf([]string{client.MeshObjectKind.Tenant, client.MeshObjectKind.Workspace}...),
								},
								PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
							},
							"uuid": schema.StringAttribute{
								MarkdownDescription: "UUID of the target workspace or tenant.",
								Optional:            true,
								Validators: []validator.String{stringvalidator.ExactlyOneOf(
									path.MatchRelative().AtParent().AtName("uuid"),
									path.MatchRelative().AtParent().AtName("identifier"),
								)},
								PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
							},
							"identifier": schema.StringAttribute{
								MarkdownDescription: "Identifier of the target workspace.",
								Optional:            true,
								PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
							},
						},
					},
					"inputs": schema.MapNestedAttribute{
						MarkdownDescription: "App-team inputs (`USER_INPUT` in the referenced definition version). Set either `value` (plain string or `jsonencode(...)`) or `sensitive` for each input key.",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"value": schema.StringAttribute{
									MarkdownDescription: "Non-sensitive input value. Use `jsonencode(...)` for non-string values.",
									Optional:            true,
									Validators: []validator.String{
										stringvalidator.ExactlyOneOf(
											path.MatchRelative().AtParent().AtName("value"),
											path.MatchRelative().AtParent().AtName("sensitive"),
										),
									},
								},
								"sensitive": secret.ResourceSchema(secret.ResourceSchemaOptions{
									MarkdownDescription: "Sensitive input value. Use `secret_value` for create/rotate and `secret_version` to control updates.",
									Optional:            true,
								}),
							},
						},
					},
					"inputs_platform_operator": schema.MapNestedAttribute{
						MarkdownDescription: "Platform engineer inputs (`PLATFORM_OPERATOR_MANUAL_INPUT` in the referenced definition version). Set either `value` or `sensitive` for each input key.",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"value": schema.StringAttribute{
									MarkdownDescription: "Non-sensitive input value. Use `jsonencode(...)` for non-string values.",
									Optional:            true,
									Validators: []validator.String{
										stringvalidator.ExactlyOneOf(
											path.MatchRelative().AtParent().AtName("value"),
											path.MatchRelative().AtParent().AtName("sensitive"),
										),
									},
								},
								"sensitive": secret.ResourceSchema(secret.ResourceSchemaOptions{
									MarkdownDescription: "Sensitive input value. Use `secret_value` for create/rotate and `secret_version` to control updates.",
									Optional:            true,
								}),
							},
						},
					},
					"inputs_static": schema.MapAttribute{
						MarkdownDescription: "Computed/static/system inputs from the definition version (all assignment types except `USER_INPUT` and `PLATFORM_OPERATOR_MANUAL_INPUT`).",
						ElementType:         types.StringType,
						Computed:            true,
						PlanModifiers:       []planmodifier.Map{mapplanmodifier.UseStateForUnknown()},
					},
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
						PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
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
						MarkdownDescription: "Execution status of the building block.",
						Computed:            true,
					},
					"force_purge": schema.BoolAttribute{
						MarkdownDescription: "Indicates whether an operator has requested purging of this Building Block.",
						Computed:            true,
					},
					"outputs": buildingBlockOutputs(),
					"latest_run": schema.SingleNestedAttribute{
						MarkdownDescription: "Latest building block run, derived from the meshBuildingBlockRun list endpoint filtered by `buildingBlockUuid`.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"uuid": schema.StringAttribute{
								Computed: true,
							},
							"run_number": schema.Int64Attribute{
								Computed: true,
							},
							"status": schema.StringAttribute{
								Computed: true,
							},
							"behavior": schema.StringAttribute{
								Computed: true,
							},
						},
					},
				},
			},
			"retrigger_run": schema.StringAttribute{
				MarkdownDescription: "Change this value to explicitly retrigger a building block run.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"content_hash": schema.StringAttribute{
				MarkdownDescription: "Set this to a building block definition version `content_hash` (for example `version_latest.content_hash` or `version_latest_release.content_hash`) to retrigger a run when the referenced definition content changes.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"wait_for_completion": schema.BoolAttribute{
				MarkdownDescription: "Whether to wait for the building block to reach a terminal state (SUCCEEDED or FAILED) before completing create/update/delete operations. The provider emits actionable warnings if the run is blocked in `WAITING_FOR_OPERATOR_INPUT`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"purge_on_delete": schema.BoolAttribute{
				MarkdownDescription: "When true, deletes with purge mode (`mode=PURGE`) instead of regular delete mode. This is a last resort option for stuck deletions.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
	}
}

func (r *buildingBlockV3Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	bb := client.MeshBuildingBlockV3Create{
		Spec: client.MeshBuildingBlockV3Spec{
			ParentBuildingBlocks:              make([]client.MeshBuildingBlockParent, 0),
			BuildingBlockDefinitionVersionRef: client.MeshBuildingBlockV2DefinitionVersionRef{},
			TargetRef:                         client.MeshBuildingBlockV2TargetRef{},
			Inputs:                            make(map[string]client.MeshBuildingBlockV3InputValue),
			InputsPlatformOperator:            make(map[string]client.MeshBuildingBlockV3InputValue),
			InputsStatic:                      make(map[string]client.MeshBuildingBlockV3InputValue),
		},
	}

	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("display_name"), &bb.Spec.DisplayName)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("building_block_definition_version_ref"), &bb.Spec.BuildingBlockDefinitionVersionRef)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("parent_building_blocks"), &bb.Spec.ParentBuildingBlocks)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("target_ref"), &bb.Spec.TargetRef)...)

	var userInputs map[string]buildingBlockV3InputModel
	var operatorInputs map[string]buildingBlockV3InputModel
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("inputs"), &userInputs)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("inputs_platform_operator"), &operatorInputs)...)

	var waitForCompletion types.Bool
	var purgeOnDelete types.Bool
	var retriggerRun types.String
	var contentHash types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("wait_for_completion"), &waitForCompletion)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("purge_on_delete"), &purgeOnDelete)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("retrigger_run"), &retriggerRun)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("content_hash"), &contentHash)...)

	if resp.Diagnostics.HasError() {
		return
	}
	waitForCompletionValue := true
	if !waitForCompletion.IsNull() && !waitForCompletion.IsUnknown() {
		waitForCompletionValue = waitForCompletion.ValueBool()
	}

	inputAssignments := r.loadInputAssignmentsByDefinitionVersionUUID(
		ctx,
		bb.Spec.BuildingBlockDefinitionVersionRef.Uuid,
		path.Root("spec").AtName("building_block_definition_version_ref").AtName("uuid"),
		&resp.Diagnostics,
	)
	if resp.Diagnostics.HasError() {
		return
	}
	validateConfiguredInputAssignments(&resp.Diagnostics, path.Root("spec").AtName("inputs"), userInputs, inputAssignments, buildingBlockV3InputBucketUser)
	validateConfiguredInputAssignments(&resp.Diagnostics, path.Root("spec").AtName("inputs_platform_operator"), operatorInputs, inputAssignments, buildingBlockV3InputBucketPlatformOperator)
	addMissingPlatformOperatorInputWarning(&resp.Diagnostics, inputAssignments, operatorInputs)
	if resp.Diagnostics.HasError() {
		return
	}

	bb.Spec.Inputs = mapInputModelsToClientValues(
		ctx,
		&resp.Diagnostics,
		req.Config,
		req.Plan,
		nil,
		path.Root("spec").AtName("inputs"),
		userInputs,
		inputAssignments,
	)
	bb.Spec.InputsPlatformOperator = mapInputModelsToClientValues(
		ctx,
		&resp.Diagnostics,
		req.Config,
		req.Plan,
		nil,
		path.Root("spec").AtName("inputs_platform_operator"),
		operatorInputs,
		inputAssignments,
	)
	bb.Spec.Inputs = mergeClientInputValues(bb.Spec.Inputs, bb.Spec.InputsPlatformOperator)
	bb.Spec.InputsPlatformOperator = nil
	bb.Spec.InputsStatic = nil
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.meshBuildingBlockV3Client.Create(ctx, &bb)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating building block",
			"Could not create building block, unexpected error: "+err.Error(),
		)
		return
	}

	createdInputs, createdOperatorInputs, createdStaticInputs := splitInputValuesByAssignment(&resp.Diagnostics, inputAssignments, created.Spec)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.setStateFromResponseV3WithLatestRun(ctx, &resp.State, created)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("inputs"), mergeConfiguredInputModels(createdInputs, userInputs))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("inputs_platform_operator"), mergeConfiguredInputModels(createdOperatorInputs, operatorInputs))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("inputs_static"), createdStaticInputs)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("wait_for_completion"), types.BoolValue(waitForCompletionValue))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("purge_on_delete"), purgeOnDelete)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("retrigger_run"), retriggerRun)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("content_hash"), contentHash)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if waitForCompletionValue {
		var lastPollOutput *client.MeshBuildingBlockV3
		err := poll.AtMostFor(30*time.Minute, r.meshBuildingBlockV3Client.ReadFunc(created.Metadata.Uuid), poll.WithLastResultTo(&lastPollOutput)).
			Until(ctx, (*client.MeshBuildingBlockV3).CreateSuccessful)
		if lastPollOutput != nil {
			polledInputs, polledOperatorInputs, polledStaticInputs := splitInputValuesByAssignment(&resp.Diagnostics, inputAssignments, lastPollOutput.Spec)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.Diagnostics.Append(r.setStateFromResponseV3WithLatestRun(ctx, &resp.State, lastPollOutput)...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("inputs"), mergeConfiguredInputModels(polledInputs, userInputs))...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("inputs_platform_operator"), mergeConfiguredInputModels(polledOperatorInputs, operatorInputs))...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("inputs_static"), polledStaticInputs)...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("wait_for_completion"), types.BoolValue(waitForCompletionValue))...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("purge_on_delete"), purgeOnDelete)...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("retrigger_run"), retriggerRun)...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("content_hash"), contentHash)...)
		}
		if err != nil {
			if lastPollOutput != nil && lastPollOutput.Status.Status == client.BUILDING_BLOCK_STATUS_WAITING_FOR_OPERATOR_INPUT {
				addWaitingForOperatorInputWarning(&resp.Diagnostics, path.Root("spec").AtName("inputs_platform_operator"), operatorInputs)
				resp.Diagnostics.AddError(
					"Building block creation blocked by operator input",
					"wait_for_completion is true, but the building block remains in WAITING_FOR_OPERATOR_INPUT. Provide all required values in `spec.inputs_platform_operator` using a principal with platform engineer permissions, or set `wait_for_completion = false` and continue once an operator has provided the missing input in meshStack.",
				)
				return
			}
			resp.Diagnostics.AddError("Failed to await building block creation", err.Error())
			return
		}
	} else if created.Status.Status == client.BUILDING_BLOCK_STATUS_WAITING_FOR_OPERATOR_INPUT {
		addWaitingForOperatorInputWarning(&resp.Diagnostics, path.Root("spec").AtName("inputs_platform_operator"), operatorInputs)
	}
}

func (r *buildingBlockV3Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var uuid string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)
	if resp.Diagnostics.HasError() {
		return
	}

	waitForCompletion := types.BoolValue(true)
	purgeOnDelete := types.BoolValue(false)
	var currentWaitForCompletion types.Bool
	var currentPurgeOnDelete types.Bool
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("wait_for_completion"), &currentWaitForCompletion)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("purge_on_delete"), &currentPurgeOnDelete)...)
	if !currentWaitForCompletion.IsNull() && !currentWaitForCompletion.IsUnknown() {
		waitForCompletion = currentWaitForCompletion
	}
	if !currentPurgeOnDelete.IsNull() && !currentPurgeOnDelete.IsUnknown() {
		purgeOnDelete = currentPurgeOnDelete
	}

	retriggerRun := types.StringValue("")
	contentHash := types.StringValue("")
	var currentRetriggerRun types.String
	var currentContentHash types.String
	var currentInputs map[string]buildingBlockV3InputModel
	var currentOperatorInputs map[string]buildingBlockV3InputModel
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("retrigger_run"), &currentRetriggerRun)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("content_hash"), &currentContentHash)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("spec").AtName("inputs"), &currentInputs)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("spec").AtName("inputs_platform_operator"), &currentOperatorInputs)...)
	if !currentRetriggerRun.IsNull() && !currentRetriggerRun.IsUnknown() {
		retriggerRun = currentRetriggerRun
	}
	if !currentContentHash.IsNull() && !currentContentHash.IsUnknown() {
		contentHash = currentContentHash
	}
	if resp.Diagnostics.HasError() {
		return
	}

	bb, err := r.meshBuildingBlockV3Client.Read(ctx, uuid)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read building block", err.Error())
		return
	}

	if bb == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	inputAssignments := r.loadInputAssignmentsByDefinitionVersionUUID(
		ctx,
		bb.Spec.BuildingBlockDefinitionVersionRef.Uuid,
		path.Root("spec").AtName("building_block_definition_version_ref").AtName("uuid"),
		&resp.Diagnostics,
	)
	if resp.Diagnostics.HasError() {
		return
	}
	readInputs, readOperatorInputs, readStaticInputs := splitInputValuesByAssignment(&resp.Diagnostics, inputAssignments, bb.Spec)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.setStateFromResponseV3WithLatestRun(ctx, &resp.State, bb)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("inputs"), normalizeReadInputModels(currentInputs, readInputs))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("inputs_platform_operator"), normalizeReadInputModels(currentOperatorInputs, readOperatorInputs))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("inputs_static"), readStaticInputs)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("wait_for_completion"), waitForCompletion)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("purge_on_delete"), purgeOnDelete)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("retrigger_run"), retriggerRun)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("content_hash"), contentHash)...)
}

func (r *buildingBlockV3Resource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}

	var planInputs map[string]buildingBlockV3InputModel
	var planOperatorInputs map[string]buildingBlockV3InputModel
	var stateInputs map[string]buildingBlockV3InputModel
	var stateOperatorInputs map[string]buildingBlockV3InputModel
	var planRetriggerRun types.String
	var stateRetriggerRun types.String
	var planContentHash types.String
	var stateContentHash types.String

	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("inputs"), &planInputs)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("inputs_platform_operator"), &planOperatorInputs)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("spec").AtName("inputs"), &stateInputs)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("spec").AtName("inputs_platform_operator"), &stateOperatorInputs)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("retrigger_run"), &planRetriggerRun)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("retrigger_run"), &stateRetriggerRun)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("content_hash"), &planContentHash)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("content_hash"), &stateContentHash)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if reflect.DeepEqual(planInputs, stateInputs) &&
		reflect.DeepEqual(planOperatorInputs, stateOperatorInputs) &&
		planRetriggerRun.Equal(stateRetriggerRun) &&
		planContentHash.Equal(stateContentHash) {
		return
	}

	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("status").AtName("status"), types.StringUnknown())...)
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("status").AtName("latest_run"), types.ObjectUnknown(map[string]attr.Type{
		"uuid":       types.StringType,
		"run_number": types.Int64Type,
		"status":     types.StringType,
		"behavior":   types.StringType,
	}))...)
}

func (r *buildingBlockV3Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var uuid string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)
	if resp.Diagnostics.HasError() {
		return
	}

	update := client.MeshBuildingBlockV3Create{
		Spec: client.MeshBuildingBlockV3Spec{
			ParentBuildingBlocks:              make([]client.MeshBuildingBlockParent, 0),
			BuildingBlockDefinitionVersionRef: client.MeshBuildingBlockV2DefinitionVersionRef{},
			TargetRef:                         client.MeshBuildingBlockV2TargetRef{},
			Inputs:                            make(map[string]client.MeshBuildingBlockV3InputValue),
			InputsPlatformOperator:            make(map[string]client.MeshBuildingBlockV3InputValue),
			InputsStatic:                      make(map[string]client.MeshBuildingBlockV3InputValue),
		},
	}

	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("display_name"), &update.Spec.DisplayName)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("building_block_definition_version_ref"), &update.Spec.BuildingBlockDefinitionVersionRef)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("parent_building_blocks"), &update.Spec.ParentBuildingBlocks)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("target_ref"), &update.Spec.TargetRef)...)
	var stateDisplayName string
	var stateDefinitionVersionRef client.MeshBuildingBlockV2DefinitionVersionRef
	var stateParentBuildingBlocks []client.MeshBuildingBlockParent
	var stateTargetRef client.MeshBuildingBlockV2TargetRef
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("spec").AtName("display_name"), &stateDisplayName)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("spec").AtName("building_block_definition_version_ref"), &stateDefinitionVersionRef)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("spec").AtName("parent_building_blocks"), &stateParentBuildingBlocks)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("spec").AtName("target_ref"), &stateTargetRef)...)

	var planInputs map[string]buildingBlockV3InputModel
	var planOperatorInputs map[string]buildingBlockV3InputModel
	var stateInputs map[string]buildingBlockV3InputModel
	var stateOperatorInputs map[string]buildingBlockV3InputModel
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("inputs"), &planInputs)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("inputs_platform_operator"), &planOperatorInputs)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("spec").AtName("inputs"), &stateInputs)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("spec").AtName("inputs_platform_operator"), &stateOperatorInputs)...)

	var planRetriggerRun types.String
	var stateRetriggerRun types.String
	var planContentHash types.String
	var stateContentHash types.String
	var waitForCompletion types.Bool
	var purgeOnDelete types.Bool
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("retrigger_run"), &planRetriggerRun)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("retrigger_run"), &stateRetriggerRun)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("content_hash"), &planContentHash)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("content_hash"), &stateContentHash)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("wait_for_completion"), &waitForCompletion)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("purge_on_delete"), &purgeOnDelete)...)
	if resp.Diagnostics.HasError() {
		return
	}
	waitForCompletionValue := true
	if !waitForCompletion.IsNull() && !waitForCompletion.IsUnknown() {
		waitForCompletionValue = waitForCompletion.ValueBool()
	}

	inputAssignments := r.loadInputAssignmentsByDefinitionVersionUUID(
		ctx,
		update.Spec.BuildingBlockDefinitionVersionRef.Uuid,
		path.Root("spec").AtName("building_block_definition_version_ref").AtName("uuid"),
		&resp.Diagnostics,
	)
	if resp.Diagnostics.HasError() {
		return
	}
	validateConfiguredInputAssignments(&resp.Diagnostics, path.Root("spec").AtName("inputs"), planInputs, inputAssignments, buildingBlockV3InputBucketUser)
	validateConfiguredInputAssignments(&resp.Diagnostics, path.Root("spec").AtName("inputs_platform_operator"), planOperatorInputs, inputAssignments, buildingBlockV3InputBucketPlatformOperator)
	addMissingPlatformOperatorInputWarning(&resp.Diagnostics, inputAssignments, planOperatorInputs)
	if resp.Diagnostics.HasError() {
		return
	}

	update.Spec.Inputs = mapInputModelsToClientValues(
		ctx,
		&resp.Diagnostics,
		req.Config,
		req.Plan,
		req.State,
		path.Root("spec").AtName("inputs"),
		planInputs,
		inputAssignments,
	)
	update.Spec.InputsPlatformOperator = mapInputModelsToClientValues(
		ctx,
		&resp.Diagnostics,
		req.Config,
		req.Plan,
		req.State,
		path.Root("spec").AtName("inputs_platform_operator"),
		planOperatorInputs,
		inputAssignments,
	)
	update.Spec.Inputs = mergeClientInputValues(update.Spec.Inputs, update.Spec.InputsPlatformOperator)
	update.Spec.InputsPlatformOperator = nil
	update.Spec.InputsStatic = nil
	if resp.Diagnostics.HasError() {
		return
	}

	specChanged := update.Spec.DisplayName != stateDisplayName ||
		!reflect.DeepEqual(update.Spec.BuildingBlockDefinitionVersionRef, stateDefinitionVersionRef) ||
		!reflect.DeepEqual(update.Spec.ParentBuildingBlocks, stateParentBuildingBlocks) ||
		!reflect.DeepEqual(update.Spec.TargetRef, stateTargetRef)
	inputsChanged := !reflect.DeepEqual(planInputs, stateInputs) || !reflect.DeepEqual(planOperatorInputs, stateOperatorInputs)
	updateChanged := specChanged || inputsChanged
	retriggerChanged := !planRetriggerRun.Equal(stateRetriggerRun)
	contentHashChanged := !planContentHash.Equal(stateContentHash)

	if updateChanged {
		if _, err := r.meshBuildingBlockV3Client.Update(ctx, uuid, &update); err != nil {
			resp.Diagnostics.AddError("Error updating building block", err.Error())
			return
		}
	}

	if retriggerChanged || contentHashChanged {
		if _, err := r.meshBuildingBlockV3Client.RetriggerRun(ctx, uuid); err != nil {
			resp.Diagnostics.AddError("Error retriggering building block run", err.Error())
			return
		}
	}

	if waitForCompletionValue && (updateChanged || retriggerChanged || contentHashChanged) {
		var lastPollOutput *client.MeshBuildingBlockV3
		err := poll.AtMostFor(30*time.Minute, r.meshBuildingBlockV3Client.ReadFunc(uuid), poll.WithLastResultTo(&lastPollOutput)).
			Until(ctx, (*client.MeshBuildingBlockV3).CreateSuccessful)
		if lastPollOutput != nil {
			polledInputs, polledOperatorInputs, polledStaticInputs := splitInputValuesByAssignment(&resp.Diagnostics, inputAssignments, lastPollOutput.Spec)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.Diagnostics.Append(r.setStateFromResponseV3WithLatestRun(ctx, &resp.State, lastPollOutput)...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("inputs"), mergeConfiguredInputModels(polledInputs, planInputs))...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("inputs_platform_operator"), mergeConfiguredInputModels(polledOperatorInputs, planOperatorInputs))...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("inputs_static"), polledStaticInputs)...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("wait_for_completion"), types.BoolValue(waitForCompletionValue))...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("purge_on_delete"), purgeOnDelete)...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("retrigger_run"), planRetriggerRun)...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("content_hash"), planContentHash)...)
		}
		if err != nil {
			if lastPollOutput != nil && lastPollOutput.Status.Status == client.BUILDING_BLOCK_STATUS_WAITING_FOR_OPERATOR_INPUT {
				addWaitingForOperatorInputWarning(&resp.Diagnostics, path.Root("spec").AtName("inputs_platform_operator"), planOperatorInputs)
				resp.Diagnostics.AddError(
					"Building block update blocked by operator input",
					"wait_for_completion is true, but the building block remains in WAITING_FOR_OPERATOR_INPUT. Provide all required values in `spec.inputs_platform_operator` using a principal with platform engineer permissions, or set `wait_for_completion = false` and continue once an operator has provided the missing input in meshStack.",
				)
				return
			}
			resp.Diagnostics.AddError("Failed to await building block update", err.Error())
			return
		}
		return
	}

	current, err := r.meshBuildingBlockV3Client.Read(ctx, uuid)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read building block after update", err.Error())
		return
	}
	if current == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	currentInputsRemote, currentOperatorInputsRemote, currentStaticInputs := splitInputValuesByAssignment(&resp.Diagnostics, inputAssignments, current.Spec)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.setStateFromResponseV3WithLatestRun(ctx, &resp.State, current)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("inputs"), mergeConfiguredInputModels(currentInputsRemote, planInputs))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("inputs_platform_operator"), mergeConfiguredInputModels(currentOperatorInputsRemote, planOperatorInputs))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("inputs_static"), currentStaticInputs)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("wait_for_completion"), types.BoolValue(waitForCompletionValue))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("purge_on_delete"), purgeOnDelete)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("retrigger_run"), planRetriggerRun)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("content_hash"), planContentHash)...)
	if current.Status.Status == client.BUILDING_BLOCK_STATUS_WAITING_FOR_OPERATOR_INPUT {
		addWaitingForOperatorInputWarning(&resp.Diagnostics, path.Root("spec").AtName("inputs_platform_operator"), planOperatorInputs)
	}
}

func (r *buildingBlockV3Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var uuid string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var waitForCompletion types.Bool
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("wait_for_completion"), &waitForCompletion)...)
	var purgeOnDelete types.Bool
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("purge_on_delete"), &purgeOnDelete)...)
	if resp.Diagnostics.HasError() {
		return
	}
	waitForCompletionValue := true
	if !waitForCompletion.IsNull() && !waitForCompletion.IsUnknown() {
		waitForCompletionValue = waitForCompletion.ValueBool()
	}

	if err := r.meshBuildingBlockV3Client.Delete(ctx, uuid, purgeOnDelete.ValueBool()); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting building block",
			"Could not delete building block, unexpected error: "+err.Error(),
		)
		return
	}

	if waitForCompletionValue {
		if err := poll.AtMostFor(30*time.Minute, r.meshBuildingBlockV3Client.ReadFunc(uuid)).
			Until(ctx, (*client.MeshBuildingBlockV3).DeletionSuccessful); err != nil {
			resp.Diagnostics.AddError("Failed to await building block deletion", err.Error())
			return
		}
	}
}

func (r *buildingBlockV3Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("metadata").AtName("uuid"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("wait_for_completion"), types.BoolValue(true))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("purge_on_delete"), types.BoolValue(false))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("retrigger_run"), types.StringValue(""))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("content_hash"), types.StringValue(""))...)
}

func (r *buildingBlockV3Resource) MoveState(ctx context.Context) []resource.StateMover {
	return []resource.StateMover{
		{
			SourceSchema: buildingBlockV1MoveStateSchema(ctx),
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				if req.SourceTypeName != "meshstack_buildingblock" {
					return
				}
				if req.SourceState == nil {
					return
				}

				var source buildingBlockV1MoveStateModel
				resp.Diagnostics.Append(req.SourceState.Get(ctx, &source)...)
				if resp.Diagnostics.HasError() {
					return
				}

				var targetIdentifier *string
				if !source.Metadata.TenantIdentifier.IsNull() && !source.Metadata.TenantIdentifier.IsUnknown() {
					targetIdentifier = source.Metadata.TenantIdentifier.ValueStringPointer()
				}

				var markedForDeletionOn *string
				if !source.Metadata.MarkedForDeletionOn.IsNull() && !source.Metadata.MarkedForDeletionOn.IsUnknown() {
					markedForDeletionOn = source.Metadata.MarkedForDeletionOn.ValueStringPointer()
				}

				var markedForDeletionBy *string
				if !source.Metadata.MarkedForDeletionBy.IsNull() && !source.Metadata.MarkedForDeletionBy.IsUnknown() {
					markedForDeletionBy = source.Metadata.MarkedForDeletionBy.ValueStringPointer()
				}

				target := buildingBlockV3MoveStateModel{
					Metadata: client.MeshBuildingBlockV2Metadata{
						Uuid:                source.Metadata.Uuid.ValueString(),
						OwnedByWorkspace:    "",
						CreatedOn:           source.Metadata.CreatedOn.ValueString(),
						MarkedForDeletionOn: markedForDeletionOn,
						MarkedForDeletionBy: markedForDeletionBy,
					},
					Spec: buildingBlockV3MoveStateSpecModel{
						DisplayName: source.Spec.DisplayName.ValueString(),
						BuildingBlockDefinitionVersionRef: client.MeshBuildingBlockV2DefinitionVersionRef{
							Uuid: "",
						},
						TargetRef: client.MeshBuildingBlockV2TargetRef{
							Kind:       client.MeshObjectKind.Tenant,
							Identifier: targetIdentifier,
						},
						Inputs:                 make(map[string]buildingBlockV3InputModel, len(source.Spec.Inputs)),
						InputsPlatformOperator: map[string]buildingBlockV3InputModel{},
						InputsStatic:           map[string]string{},
						ParentBuildingBlocks:   source.Spec.ParentBuildingBlocks,
					},
					Status: buildingBlockV3MoveStateStatusModel{
						Status:     "",
						ForcePurge: false,
						Outputs:    source.Status.Outputs,
						LatestRun:  nil,
					},
					RetriggerRun:      types.StringValue(""),
					ContentHash:       types.StringValue(""),
					WaitForCompletion: types.BoolValue(true),
					PurgeOnDelete:     types.BoolValue(false),
				}

				if !source.Status.Status.IsNull() && !source.Status.Status.IsUnknown() {
					target.Status.Status = source.Status.Status.ValueString()
				}
				if !source.Metadata.ForcePurge.IsNull() && !source.Metadata.ForcePurge.IsUnknown() {
					target.Status.ForcePurge = source.Metadata.ForcePurge.ValueBool()
				}

				for key, input := range source.Spec.Inputs {
					value, _ := input.extractIoValue()
					if value == nil {
						continue
					}
					target.Spec.Inputs[key] = buildingBlockV3InputModel{
						Value: types.StringValue(clientInputValueToString(client.MeshBuildingBlockV3InputValue{Value: value})),
					}
				}

				resp.Diagnostics.Append(resp.TargetState.Set(ctx, target)...)
			},
		},
		{
			SourceSchema: buildingBlockV2MoveStateSchema(ctx),
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				if req.SourceTypeName != "meshstack_building_block_v2" {
					return
				}
				if req.SourceState == nil {
					return
				}

				var source buildingBlockV2MoveStateModel
				resp.Diagnostics.Append(req.SourceState.Get(ctx, &source)...)
				if resp.Diagnostics.HasError() {
					return
				}

				target := buildingBlockV3MoveStateModel{
					Metadata: source.Metadata,
					Spec: buildingBlockV3MoveStateSpecModel{
						DisplayName:                       source.Spec.DisplayName,
						BuildingBlockDefinitionVersionRef: source.Spec.BuildingBlockDefinitionVersionRef,
						TargetRef:                         source.Spec.TargetRef,
						Inputs:                            make(map[string]buildingBlockV3InputModel, len(source.Spec.Inputs)),
						InputsPlatformOperator:            map[string]buildingBlockV3InputModel{},
						InputsStatic:                      map[string]string{},
						ParentBuildingBlocks:              source.Spec.ParentBuildingBlocks,
					},
					Status: buildingBlockV3MoveStateStatusModel{
						Status:     "",
						ForcePurge: false,
						Outputs:    source.Status.Outputs,
						LatestRun:  nil,
					},
					RetriggerRun:      types.StringValue(""),
					ContentHash:       types.StringValue(""),
					WaitForCompletion: source.WaitForCompletion,
					PurgeOnDelete:     types.BoolValue(false),
				}
				if !source.Status.Status.IsNull() && !source.Status.Status.IsUnknown() {
					target.Status.Status = source.Status.Status.ValueString()
				}
				if !source.Status.ForcePurge.IsNull() && !source.Status.ForcePurge.IsUnknown() {
					target.Status.ForcePurge = source.Status.ForcePurge.ValueBool()
				}

				for key, input := range source.Spec.Inputs {
					value, _ := input.extractIoValue()
					if value == nil {
						continue
					}
					target.Spec.Inputs[key] = buildingBlockV3InputModel{
						Value: types.StringValue(clientInputValueToString(client.MeshBuildingBlockV3InputValue{Value: value})),
					}
				}

				resp.Diagnostics.Append(resp.TargetState.Set(ctx, target)...)
			},
		},
	}
}

func buildingBlockV1MoveStateSchema(ctx context.Context) *schema.Schema {
	v1 := &buildingBlockResource{}
	resp := &resource.SchemaResponse{}
	v1.Schema(ctx, resource.SchemaRequest{}, resp)
	return &resp.Schema
}

func buildingBlockV2MoveStateSchema(ctx context.Context) *schema.Schema {
	v2 := &buildingBlockV2Resource{}
	resp := &resource.SchemaResponse{}
	v2.Schema(ctx, resource.SchemaRequest{}, resp)
	return &resp.Schema
}

type buildingBlockV1MoveStateModel struct {
	Metadata buildingBlockV1MoveStateMetadataModel `tfsdk:"metadata"`
	Spec     buildingBlockV1MoveStateSpecModel     `tfsdk:"spec"`
	Status   buildingBlockV1MoveStateStatusModel   `tfsdk:"status"`
}

type buildingBlockV1MoveStateMetadataModel struct {
	Uuid                types.String `tfsdk:"uuid"`
	DefinitionUuid      types.String `tfsdk:"definition_uuid"`
	DefinitionVersion   types.Int64  `tfsdk:"definition_version"`
	TenantIdentifier    types.String `tfsdk:"tenant_identifier"`
	ForcePurge          types.Bool   `tfsdk:"force_purge"`
	CreatedOn           types.String `tfsdk:"created_on"`
	MarkedForDeletionOn types.String `tfsdk:"marked_for_deletion_on"`
	MarkedForDeletionBy types.String `tfsdk:"marked_for_deletion_by"`
}

type buildingBlockV1MoveStateSpecModel struct {
	DisplayName          types.String                           `tfsdk:"display_name"`
	ParentBuildingBlocks []client.MeshBuildingBlockParent       `tfsdk:"parent_building_blocks"`
	Inputs               map[string]buildingBlockUserInputModel `tfsdk:"inputs"`
	CombinedInputs       types.Map                              `tfsdk:"combined_inputs"`
}

type buildingBlockV1MoveStateStatusModel struct {
	Status  types.String                        `tfsdk:"status"`
	Outputs map[string]buildingBlockOutputModel `tfsdk:"outputs"`
}

type buildingBlockV2MoveStateModel struct {
	Metadata          client.MeshBuildingBlockV2Metadata  `tfsdk:"metadata"`
	Spec              buildingBlockV2MoveStateSpecModel   `tfsdk:"spec"`
	Status            buildingBlockV2MoveStateStatusModel `tfsdk:"status"`
	WaitForCompletion types.Bool                          `tfsdk:"wait_for_completion"`
}

type buildingBlockV2MoveStateSpecModel struct {
	DisplayName                       string                                         `tfsdk:"display_name"`
	BuildingBlockDefinitionVersionRef client.MeshBuildingBlockV2DefinitionVersionRef `tfsdk:"building_block_definition_version_ref"`
	TargetRef                         client.MeshBuildingBlockV2TargetRef            `tfsdk:"target_ref"`
	Inputs                            map[string]buildingBlockUserInputModel         `tfsdk:"inputs"`
	CombinedInputs                    map[string]buildingBlockIoModel                `tfsdk:"combined_inputs"`
	ParentBuildingBlocks              []client.MeshBuildingBlockParent               `tfsdk:"parent_building_blocks"`
}

type buildingBlockV2MoveStateStatusModel struct {
	Status     types.String                        `tfsdk:"status"`
	ForcePurge types.Bool                          `tfsdk:"force_purge"`
	Outputs    map[string]buildingBlockOutputModel `tfsdk:"outputs"`
}

type buildingBlockV3MoveStateModel struct {
	Metadata          client.MeshBuildingBlockV2Metadata  `tfsdk:"metadata"`
	Spec              buildingBlockV3MoveStateSpecModel   `tfsdk:"spec"`
	Status            buildingBlockV3MoveStateStatusModel `tfsdk:"status"`
	RetriggerRun      types.String                        `tfsdk:"retrigger_run"`
	ContentHash       types.String                        `tfsdk:"content_hash"`
	WaitForCompletion types.Bool                          `tfsdk:"wait_for_completion"`
	PurgeOnDelete     types.Bool                          `tfsdk:"purge_on_delete"`
}

type buildingBlockV3MoveStateSpecModel struct {
	DisplayName                       string                                         `tfsdk:"display_name"`
	BuildingBlockDefinitionVersionRef client.MeshBuildingBlockV2DefinitionVersionRef `tfsdk:"building_block_definition_version_ref"`
	TargetRef                         client.MeshBuildingBlockV2TargetRef            `tfsdk:"target_ref"`
	Inputs                            map[string]buildingBlockV3InputModel           `tfsdk:"inputs"`
	InputsPlatformOperator            map[string]buildingBlockV3InputModel           `tfsdk:"inputs_platform_operator"`
	InputsStatic                      map[string]string                              `tfsdk:"inputs_static"`
	ParentBuildingBlocks              []client.MeshBuildingBlockParent               `tfsdk:"parent_building_blocks"`
}

type buildingBlockV3MoveStateStatusModel struct {
	Status     string                              `tfsdk:"status"`
	ForcePurge bool                                `tfsdk:"force_purge"`
	Outputs    map[string]buildingBlockOutputModel `tfsdk:"outputs"`
	LatestRun  *buildingBlockV3LatestRunModel      `tfsdk:"latest_run"`
}

type buildingBlockV3LatestRunModel struct {
	Uuid      string `tfsdk:"uuid"`
	RunNumber int64  `tfsdk:"run_number"`
	Status    string `tfsdk:"status"`
	Behavior  string `tfsdk:"behavior"`
}

type buildingBlockV3InputModel struct {
	Value     types.String   `tfsdk:"value"`
	Sensitive *secret.Secret `tfsdk:"sensitive"`
}

type buildingBlockV3InputBucket string

const (
	buildingBlockV3InputBucketUser             buildingBlockV3InputBucket = "user"
	buildingBlockV3InputBucketPlatformOperator buildingBlockV3InputBucket = "platform_operator"
	buildingBlockV3InputBucketStatic           buildingBlockV3InputBucket = "static"
)

type buildingBlockV3InputAssignment struct {
	AssignmentType client.MeshBuildingBlockInputAssignmentType
	Bucket         buildingBlockV3InputBucket
	ValueType      string
}

func (r *buildingBlockV3Resource) loadInputAssignmentsByDefinitionVersionUUID(
	ctx context.Context,
	definitionVersionUUID string,
	attributePath path.Path,
	diags *diag.Diagnostics,
) map[string]buildingBlockV3InputAssignment {
	if definitionVersionUUID == "" {
		diags.AddAttributeError(
			attributePath,
			"Missing building block definition version UUID",
			"Cannot classify building block inputs without `spec.building_block_definition_version_ref.uuid`.",
		)
		return nil
	}

	version, err := r.meshBuildingBlockDefinitionVersionClient.Read(ctx, definitionVersionUUID)
	if err != nil {
		diags.AddAttributeError(
			attributePath,
			"Unable to read building block definition version",
			fmt.Sprintf("Could not read building block definition version %q: %s", definitionVersionUUID, err.Error()),
		)
		return nil
	}
	if version == nil {
		diags.AddAttributeError(
			attributePath,
			"Building block definition version not found",
			fmt.Sprintf("No building block definition version found for UUID %q.", definitionVersionUUID),
		)
		return nil
	}

	assignments := make(map[string]buildingBlockV3InputAssignment, len(version.Spec.Inputs))
	for key, input := range version.Spec.Inputs {
		if input == nil {
			diags.AddAttributeError(
				attributePath,
				"Invalid building block definition version input metadata",
				fmt.Sprintf("Input %q in building block definition version %q has no metadata.", key, definitionVersionUUID),
			)
			continue
		}
		assignments[key] = buildingBlockV3InputAssignment{
			AssignmentType: input.AssignmentType,
			Bucket:         buildingBlockV3InputBucketFromAssignmentType(input.AssignmentType),
			ValueType:      string(input.Type),
		}
	}
	return assignments
}

func buildingBlockV3InputBucketFromAssignmentType(assignmentType client.MeshBuildingBlockInputAssignmentType) buildingBlockV3InputBucket {
	switch assignmentType {
	case client.MeshBuildingBlockInputAssignmentTypeUserInput.Unwrap():
		return buildingBlockV3InputBucketUser
	case client.MeshBuildingBlockInputAssignmentTypePlatformOperatorManualInput.Unwrap():
		return buildingBlockV3InputBucketPlatformOperator
	default:
		return buildingBlockV3InputBucketStatic
	}
}

func inputConfigPathForBucket(bucket buildingBlockV3InputBucket) string {
	switch bucket {
	case buildingBlockV3InputBucketUser:
		return "spec.inputs"
	case buildingBlockV3InputBucketPlatformOperator:
		return "spec.inputs_platform_operator"
	default:
		return "spec.inputs_static"
	}
}

func validateConfiguredInputAssignments(
	diags *diag.Diagnostics,
	attributePath path.Path,
	configuredInputs map[string]buildingBlockV3InputModel,
	assignments map[string]buildingBlockV3InputAssignment,
	expectedBucket buildingBlockV3InputBucket,
) {
	for key := range configuredInputs {
		assignment, found := assignments[key]
		if !found {
			diags.AddAttributeError(
				attributePath.AtMapKey(key),
				"Unknown input key",
				fmt.Sprintf("Input %q is not defined in the referenced building block definition version.", key),
			)
			continue
		}
		if assignment.Bucket == expectedBucket {
			continue
		}
		detail := fmt.Sprintf(
			"Input %q has assignment type `%s` in the referenced building block definition version and must be configured in `%s`.",
			key, assignment.AssignmentType, inputConfigPathForBucket(assignment.Bucket),
		)
		if assignment.Bucket == buildingBlockV3InputBucketStatic {
			detail = fmt.Sprintf(
				"Input %q has assignment type `%s` in the referenced building block definition version and cannot be configured manually. It is exposed via `spec.inputs_static`.",
				key, assignment.AssignmentType,
			)
		}
		diags.AddAttributeError(
			attributePath.AtMapKey(key),
			"Input configured in wrong attribute",
			detail,
		)
	}
}

func addMissingPlatformOperatorInputWarning(
	diags *diag.Diagnostics,
	assignments map[string]buildingBlockV3InputAssignment,
	configuredOperatorInputs map[string]buildingBlockV3InputModel,
) {
	if len(assignments) == 0 {
		return
	}

	var missingKeys []string
	for key, assignment := range assignments {
		if assignment.Bucket != buildingBlockV3InputBucketPlatformOperator {
			continue
		}
		if _, configured := configuredOperatorInputs[key]; configured {
			continue
		}
		missingKeys = append(missingKeys, key)
	}
	if len(missingKeys) == 0 {
		return
	}
	sort.Strings(missingKeys)

	diags.AddAttributeWarning(
		path.Root("spec").AtName("inputs_platform_operator"),
		"Platform operator inputs missing",
		fmt.Sprintf(
			"The definition contains platform-operator manual input(s) that are not set in `spec.inputs_platform_operator`: %s. The building block run may remain in WAITING_FOR_OPERATOR_INPUT until a platform engineer provides these values in meshStack. If you are an app team user, coordinate with your platform engineer. If you are a platform engineer, set these inputs in Terraform to avoid manual follow-up.",
			strings.Join(missingKeys, ", "),
		),
	)
}

func addWaitingForOperatorInputWarning(
	diags *diag.Diagnostics,
	attributePath path.Path,
	configuredOperatorInputs map[string]buildingBlockV3InputModel,
) {
	detail := "The building block is in WAITING_FOR_OPERATOR_INPUT. A platform engineer must provide all required operator-manual inputs before the run can continue."
	if len(configuredOperatorInputs) == 0 {
		detail += " No values were configured in `spec.inputs_platform_operator`."
	} else {
		detail += " Values are configured in `spec.inputs_platform_operator`; verify the caller has platform engineer permissions and that meshStack accepted these values."
	}
	detail += " If app teams manage this resource, coordinate with platform engineers."

	diags.AddAttributeWarning(attributePath, "Building block waiting for operator input", detail)
}

func mergeClientInputValues(inputMaps ...map[string]client.MeshBuildingBlockV3InputValue) map[string]client.MeshBuildingBlockV3InputValue {
	var merged map[string]client.MeshBuildingBlockV3InputValue
	for _, values := range inputMaps {
		if len(values) == 0 {
			continue
		}
		if merged == nil {
			merged = make(map[string]client.MeshBuildingBlockV3InputValue, len(values))
		}
		maps.Copy(merged, values)
	}
	return merged
}

func splitInputValuesByAssignment(
	diags *diag.Diagnostics,
	assignments map[string]buildingBlockV3InputAssignment,
	spec client.MeshBuildingBlockV3Spec,
) (
	inputs map[string]client.MeshBuildingBlockV3InputValue,
	inputsPlatformOperator map[string]client.MeshBuildingBlockV3InputValue,
	inputsStatic map[string]string,
) {
	combinedInputs := mergeClientInputValues(spec.Inputs, spec.InputsPlatformOperator, spec.InputsStatic)
	for key, value := range combinedInputs {
		assignment, found := assignments[key]
		if !found {
			diags.AddAttributeError(
				path.Root("spec").AtName("inputs_static").AtMapKey(key),
				"Unknown input key returned by API",
				fmt.Sprintf("Input %q is not defined in the referenced building block definition version and cannot be classified.", key),
			)
			continue
		}
		switch assignment.Bucket {
		case buildingBlockV3InputBucketUser:
			if inputs == nil {
				inputs = make(map[string]client.MeshBuildingBlockV3InputValue)
			}
			inputs[key] = value
		case buildingBlockV3InputBucketPlatformOperator:
			if inputsPlatformOperator == nil {
				inputsPlatformOperator = make(map[string]client.MeshBuildingBlockV3InputValue)
			}
			inputsPlatformOperator[key] = value
		default:
			if inputsStatic == nil {
				inputsStatic = make(map[string]string)
			}
			inputsStatic[key] = clientInputValueToString(value)
		}
	}
	return
}

func setStateFromResponseV3(ctx context.Context, state *tfsdk.State, bb *client.MeshBuildingBlockV3) (diags diag.Diagnostics) {
	diags.Append(state.SetAttribute(ctx, path.Root("metadata"), bb.Metadata)...)

	diags.Append(state.SetAttribute(ctx, path.Root("spec").AtName("display_name"), bb.Spec.DisplayName)...)
	diags.Append(state.SetAttribute(ctx, path.Root("spec").AtName("building_block_definition_version_ref"), bb.Spec.BuildingBlockDefinitionVersionRef)...)
	diags.Append(state.SetAttribute(ctx, path.Root("spec").AtName("target_ref"), bb.Spec.TargetRef)...)
	diags.Append(state.SetAttribute(ctx, path.Root("spec").AtName("parent_building_blocks"), bb.Spec.ParentBuildingBlocks)...)

	diags.Append(state.SetAttribute(ctx, path.Root("status").AtName("status"), bb.Status.Status)...)
	diags.Append(state.SetAttribute(ctx, path.Root("status").AtName("force_purge"), bb.Status.ForcePurge)...)

	outputs := make(map[string]buildingBlockOutputModel)
	for _, output := range bb.Status.Outputs {
		outputs[output.Key] = toResourceModel(output, &diags).toOutputModel()
	}
	if diags.HasError() {
		return
	}
	diags.Append(state.SetAttribute(ctx, path.Root("status").AtName("outputs"), outputs)...)

	return diags
}

func (r *buildingBlockV3Resource) setStateFromResponseV3WithLatestRun(ctx context.Context, state *tfsdk.State, bb *client.MeshBuildingBlockV3) (diags diag.Diagnostics) {
	diags.Append(setStateFromResponseV3(ctx, state, bb)...)
	if diags.HasError() {
		return
	}

	latestRun := r.deriveLatestRunFromRuns(ctx, bb.Metadata.Uuid, &diags)
	diags.Append(state.SetAttribute(ctx, path.Root("status").AtName("latest_run"), latestRun)...)
	return
}

func (r *buildingBlockV3Resource) deriveLatestRunFromRuns(ctx context.Context, buildingBlockUUID string, diags *diag.Diagnostics) *buildingBlockV3LatestRunModel {
	if buildingBlockUUID == "" || r.meshBuildingBlockRunClient == nil {
		return nil
	}

	runs, err := r.meshBuildingBlockRunClient.ListByBuildingBlockUUID(ctx, buildingBlockUUID)
	if err != nil {
		diags.AddWarning(
			"Unable to derive status.latest_run",
			fmt.Sprintf(
				"Could not list building block runs for building block %q: %s. The `status.latest_run` attribute will remain empty. Ensure the caller can list building block runs (for example `MANAGED_BUILDINGBLOCKRUN_LIST` or `ADM_BUILDINGBLOCKRUN_LIST`).",
				buildingBlockUUID,
				err.Error(),
			),
		)
		return nil
	}

	latestRun := selectLatestBuildingBlockRun(runs)
	if latestRun == nil {
		return nil
	}

	return &buildingBlockV3LatestRunModel{
		Uuid:      latestRun.Metadata.Uuid,
		RunNumber: latestRun.Spec.RunNumber,
		Status:    latestRun.Status,
		Behavior:  latestRun.Spec.Behavior,
	}
}

func selectLatestBuildingBlockRun(runs []client.MeshBuildingBlockRun) *client.MeshBuildingBlockRun {
	var latest *client.MeshBuildingBlockRun
	for i := range runs {
		candidate := &runs[i]
		if latest == nil || candidate.Spec.RunNumber > latest.Spec.RunNumber {
			latest = candidate
			continue
		}
		if candidate.Spec.RunNumber == latest.Spec.RunNumber && isCreatedAfter(candidate.Metadata.CreatedOn, latest.Metadata.CreatedOn) {
			latest = candidate
		}
	}
	return latest
}

func isCreatedAfter(a string, b string) bool {
	if a == "" {
		return false
	}
	if b == "" {
		return true
	}

	leftTime, leftErr := time.Parse(time.RFC3339Nano, a)
	rightTime, rightErr := time.Parse(time.RFC3339Nano, b)
	switch {
	case leftErr == nil && rightErr == nil:
		return leftTime.After(rightTime)
	case leftErr == nil:
		return true
	case rightErr == nil:
		return false
	default:
		return a > b
	}
}

func mapInputModelsToClientValues(
	ctx context.Context,
	diags *diag.Diagnostics,
	configGetter, planGetter, stateGetter generic.AttributeGetter,
	inputPath path.Path,
	inputs map[string]buildingBlockV3InputModel,
	assignments map[string]buildingBlockV3InputAssignment,
) map[string]client.MeshBuildingBlockV3InputValue {
	result := make(map[string]client.MeshBuildingBlockV3InputValue, len(inputs))
	for key, input := range inputs {
		assignment, hasAssignment := assignments[key]
		valueType := ""
		if hasAssignment {
			valueType = assignment.ValueType
		}

		hasValue := !input.Value.IsNull() && !input.Value.IsUnknown()
		hasSensitive := input.Sensitive != nil
		switch {
		case hasValue == hasSensitive:
			diags.AddAttributeError(
				inputPath.AtMapKey(key),
				"Invalid input configuration",
				"Each input entry must set exactly one of `value` or `sensitive`.",
			)
			continue
		case hasValue:
			result[key] = parseStringInputToClientValue(input.Value.ValueString(), valueType)
		default:
			secretValue, err := secret.ValueToConverter(ctx, configGetter, planGetter, stateGetter, inputPath.AtMapKey(key).AtName("sensitive"))
			if err != nil {
				diags.AddAttributeError(
					inputPath.AtMapKey(key).AtName("sensitive"),
					"Invalid sensitive input",
					err.Error(),
				)
				continue
			}
			result[key] = client.MeshBuildingBlockV3InputValue{
				Sensitive: &secretValue,
				ValueType: valueType,
			}
		}
	}
	return result
}

func parseStringInputToClientValue(raw string, valueType string) client.MeshBuildingBlockV3InputValue {
	return client.MeshBuildingBlockV3InputValue{
		Value:     tryDecodeJSONValue(raw),
		ValueType: valueType,
	}
}

func mapClientInputValuesToModels(inputs map[string]client.MeshBuildingBlockV3InputValue) map[string]buildingBlockV3InputModel {
	result := make(map[string]buildingBlockV3InputModel, len(inputs))
	for key, input := range inputs {
		if input.Sensitive != nil {
			var hash *string
			if input.Sensitive.Hash != nil {
				hash = input.Sensitive.Hash
			}
			result[key] = buildingBlockV3InputModel{
				Sensitive: &secret.Secret{
					Hash:    hash,
					Version: hash,
				},
			}
			continue
		}
		result[key] = buildingBlockV3InputModel{
			Value: types.StringValue(clientInputValueToString(input)),
		}
	}
	return result
}

func mergeConfiguredInputModels(
	remote map[string]client.MeshBuildingBlockV3InputValue,
	configured map[string]buildingBlockV3InputModel,
) map[string]buildingBlockV3InputModel {
	if configured == nil {
		return nil
	}

	merged := mapClientInputValuesToModels(remote)
	for key, input := range configured {
		if input.Sensitive != nil {
			continue
		}
		merged[key] = buildingBlockV3InputModel{
			Value: input.Value,
		}
	}
	return merged
}

func normalizeReadInputModels(
	current map[string]buildingBlockV3InputModel,
	remote map[string]client.MeshBuildingBlockV3InputValue,
) map[string]buildingBlockV3InputModel {
	if current == nil && len(remote) == 0 {
		return nil
	}
	return mapClientInputValuesToModels(remote)
}

func clientInputValueToString(input client.MeshBuildingBlockV3InputValue) string {
	if input.Sensitive != nil {
		if input.Sensitive.Hash != nil {
			return *input.Sensitive.Hash
		}
		if input.Sensitive.Plaintext != nil {
			return *input.Sensitive.Plaintext
		}
		return ""
	}

	switch value := input.Value.(type) {
	case nil:
		return "null"
	case string:
		return value
	default:
		out, err := json.Marshal(value)
		if err != nil {
			return fmt.Sprintf("%v", value)
		}
		return string(out)
	}
}

func tryDecodeJSONValue(raw string) any {
	var decoded any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return raw
	}
	return decoded
}
