package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	timeouts "github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"
	"github.com/meshcloud/terraform-provider-meshstack/client/types/enum"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/secret"
	"github.com/meshcloud/terraform-provider-meshstack/internal/util/poll"
	"github.com/meshcloud/terraform-provider-meshstack/internal/validators"
)

var (
	_ resource.Resource                = &buildingBlockResource{}
	_ resource.ResourceWithConfigure   = &buildingBlockResource{}
	_ resource.ResourceWithImportState = &buildingBlockResource{}
	_ resource.ResourceWithModifyPlan  = &buildingBlockResource{}
)

// defaultBuildingBlockTimeout is the fallback time to wait for a building block run to complete when
// the configuration does not set an explicit value in the `timeouts` block.
const defaultBuildingBlockTimeout = 30 * time.Minute

func NewBuildingBlockResource() resource.Resource {
	return &buildingBlockResource{}
}

type buildingBlockResource struct {
	BuildingBlockClient    client.MeshBuildingBlockV2Client
	BuildingBlockRunClient client.MeshBuildingBlockRunClient
}

func (r *buildingBlockResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_building_block"
}

func (r *buildingBlockResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		r.BuildingBlockClient = client.BuildingBlockV2
		r.BuildingBlockRunClient = client.BuildingBlockRun
	})...)
}

// parentBuildingBlocksNestedObject is the NestedAttributeObject for parent_building_blocks.
// Defined once and reused by both the Default and NestedObject fields.
var parentBuildingBlocksNestedObject = schema.NestedAttributeObject{
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
}

func (r *buildingBlockResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage a workspace or tenant building block created from a building block definition (BBD).\n\n" +
			"A building block is usually managed by the app team that owns its workspace; a platform operator " +
			"typically only creates one directly to test a draft BBD in the operator's own workspace. " +
			"Building blocks can depend on each other via `parent_building_blocks`, forming a dependency hierarchy " +
			"in which a child's inputs draw their values from a parent's outputs " +
			"(see [building block concepts](https://docs.meshcloud.io/concepts/building-block/))." + previewDisclaimer(),
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
				},
			},
			"spec": schema.SingleNestedAttribute{
				MarkdownDescription: "Building block specification.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"display_name": schema.StringAttribute{
						MarkdownDescription: "Display name for the building block as shown in meshPanel. " +
							"Changing it is applied in place (a rename) and does not trigger a building block run.",
						Required: true,
					},
					"building_block_definition_version_ref": schema.SingleNestedAttribute{
						MarkdownDescription: "References the building block definition version this building block is based on. " +
							"Changing the `uuid` upgrades the building block in place. Only upgrades to the **latest released version** " +
							"of the same definition are supported; pointing at an older or non-released version is rejected by the backend.",
						Required: true,
						Attributes: map[string]schema.Attribute{
							"uuid": schema.StringAttribute{
								MarkdownDescription: "UUID of the building block definition version. Must reference the latest released version of the definition when upgrading.",
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
							"content_hash": schema.StringAttribute{
								MarkdownDescription: "Content hash of the building block definition version. " +
									"Its purpose is to detect content changes of a **draft** BBD (whose version `uuid` stays the same) " +
									"and conveniently re-run the building block when it changes.<br>" +
									"When wired from a definition's computed `content_hash`, a change caused *only* by a hash-algorithm " +
									"version upgrade (e.g. after upgrading the provider) does **not** trigger a re-run.<br>" +
									"It is provider-only and never sent to the backend, so changing it can also be used to force a " +
									"manual re-run — use with care; with a plain workspace key (`BUILDINGBLOCK_SAVE`) this requires the " +
									"definition to have run transparency enabled (admins and the definition's platform operator are exempt).<br>" +
									"After import it is left null in state, so the first apply triggers a run if `content_hash` " +
									"is set in config. To avoid that, omit `content_hash` until after the first post-import apply, " +
									"or set it only then.",
								Optional: true,
							},
						},
					},
					"target_ref": schema.SingleNestedAttribute{
						MarkdownDescription: "References the building block target. " +
							"For `meshTenant` targets, `uuid` is required and `name` must be omitted. " +
							"For `meshWorkspace` targets, `name` is required and `uuid` must be omitted.",
						Required: true,
						Validators: []validator.Object{
							// Enforce the kind-dependent contract: meshTenant uses uuid, meshWorkspace uses name.
							validators.DiscriminatedAttributesValidator{
								Discriminator: "kind",
								RequiredFor: map[string]string{
									client.MeshObjectKind.Tenant:    "uuid",
									client.MeshObjectKind.Workspace: "name",
								},
							},
						},
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
								MarkdownDescription: "UUID of the target tenant. Required when `kind = \"meshTenant\"`, must be omitted for `kind = \"meshWorkspace\"`.",
								Optional:            true,
								Validators: []validator.String{
									// Exactly one of uuid/name must be set; the target_ref object validator
									// further enforces which one is required for the given kind.
									stringvalidator.ExactlyOneOf(
										path.MatchRelative().AtParent().AtName("uuid"),
										path.MatchRelative().AtParent().AtName("name"),
									),
								},
								PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
							},
							"name": schema.StringAttribute{
								MarkdownDescription: "Identifier of the target workspace. Required when `kind = \"meshWorkspace\"`, must be omitted for `kind = \"meshTenant\"`.",
								Optional:            true,
								PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
							},
						},
					},
					"inputs": schema.MapNestedAttribute{
						MarkdownDescription: "Input values this resource manages, keyed by input name. " +
							"Defined much like a BBD's inputs (which are richer, e.g. defaults).<br>" +
							"Set either `value` (always `jsonencode(...)`'d, including strings) or `sensitive = { secret_value = ... }`. " +
							"The `sensitive` block must be used if and only if the BBD declares the input as sensitive.<br>" +
							"App teams normally set only `USER_INPUT` inputs. `PLATFORM_OPERATOR_MANUAL_INPUT` inputs require a " +
							"platform-operator key (admin, or `MANAGED_BUILDINGBLOCK_SAVE` for the definition's owning workspace): an " +
							"operator sets them either on a block it creates from its own BBD (e.g. testing a draft), or by importing an " +
							"app-team block created from its BBD to supply the operator inputs that block is awaiting. This shared " +
							"ownership of a block (app team and operator) is experimental and will be documented more fully later; " +
							"supplying an operator input with a non-operator/non-owner key is rejected.",
						Required: true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"value": schema.StringAttribute{
									MarkdownDescription: "Non-sensitive input value, always `jsonencode(...)`'d — including strings " +
										"(e.g. `jsonencode(\"my-name\")`, `jsonencode(16)`) — to match the type the BBD declares, " +
										"the same convention as a BBD input's `argument`/`default_value`.",
									Optional: true,
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
					"parent_building_blocks": schema.SetNestedAttribute{
						Optional: true,
						Computed: true,
						MarkdownDescription: "Parent building blocks this block depends on, forming a dependency hierarchy: " +
							"a parent's outputs can feed this block's inputs, so the parents listed here should align with the " +
							"inputs that consume them (see [building block concepts](https://docs.meshcloud.io/concepts/building-block/)).<br>" +
							"Parent building blocks can only change as part of a version upgrade; changing them on their own " +
							"forces the building block to be replaced (destroyed and recreated).",
						Default:      emptySetDefault(parentBuildingBlocksNestedObject),
						NestedObject: parentBuildingBlocksNestedObject,
						PlanModifiers: []planmodifier.Set{
							setplanmodifier.RequiresReplaceIf(
								requiresReplaceParentsWhenVersionUnchanged,
								"Parent building blocks can only change as part of a version upgrade.",
								"Parent building blocks can only change as part of a version upgrade.",
							),
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
						MarkdownDescription: "Read-only: true once a purge has been requested for this building block — " +
							"via `purge_on_delete` here, or by an operator out-of-band. A purge removes the block without a " +
							"destroy run, leaving its cloud resources unmanaged (the lifecycle still reaches DELETED).",
						Computed: true,
					},
					"latest_run_uuid": schema.StringAttribute{
						MarkdownDescription: "UUID of the latest modifying (apply/destroy) run for this Building Block. " +
							"Excludes dry runs (see `latest_dry_run_uuid`). Null when no modifying run exists, or when " +
							"permissions are insufficient to read runs.",
						Computed: true,
					},
					"latest_dry_run_uuid": schema.StringAttribute{
						MarkdownDescription: "UUID of the latest dry (DETECT) run for this Building Block, but only when it is the " +
							"newest run; null otherwise. Same permission gating as `latest_run_uuid`.",
						Computed: true,
					},

					"outputs": schema.MapNestedAttribute{
						MarkdownDescription: "Outputs of building block, available after a successful run.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"value": schema.StringAttribute{
									CustomType:          jsontypes.NormalizedType{},
									MarkdownDescription: "Output value. Use `jsondecode(...)` to obtain polymorphic value depending on `value_type`.",
									Computed:            true,
								},
								"value_type": schema.StringAttribute{
									MarkdownDescription: "Data type of the value. One of " + client.MeshBuildingBlockIOTypes.Markdown() + ".",
									Computed:            true,
								},
								"assignment_type": schema.StringAttribute{
									MarkdownDescription: "How the input value is assigned. One of " + client.MeshBuildingBlockDefinitionOutputAssignmentTypes.Markdown() + ".",
									Computed:            true,
								},
							},
						},
						Computed:      true,
						PlanModifiers: []planmodifier.Map{mapplanmodifier.UseStateForUnknown()},
					},
				},
			},
			"wait_for_completion": schema.BoolAttribute{
				MarkdownDescription: "Whether to wait for the building block to reach a terminal state (SUCCEEDED or FAILED) before completing create/update operations. The provider emits actionable warnings if the run is blocked in `WAITING_FOR_OPERATOR_INPUT`. Deletion always waits for the block to be fully removed (lifecycle DELETED, or gone after a purge) regardless of this flag, so dependent resources such as the building block definition can be deleted safely afterward. Each wait is bounded by `timeouts` (see `timeouts.delete` for deprovisioning).",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"purge_on_delete": schema.BoolAttribute{
				MarkdownDescription: "When true, deletes via the `DELETE /{uuid}/purge` sub-path, which requires admin authority (`ADM_BUILDINGBLOCK_DELETE`). This is a last resort option for stuck deletions.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Update: true,
				Delete: true,
				DeleteDescription: "Maximum time to wait for the building block to finish deprovisioning after delete " +
					"(the async destroy run, or removal after a purge). On timeout the apply errors and the resource is " +
					"kept in state so the delete can be retried; it is not silently dropped. " +
					"A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration), e.g. \"30s\" or \"2h45m\".",
			}),
			"all_inputs": schema.MapNestedAttribute{
				MarkdownDescription: "Computed read-only view of **all** inputs resolved by the backend — platform-operator, " +
					"user, and static inputs (the latter derived from the BBD) — regardless of who set them.<br>" +
					"Contrast with `spec.inputs`, which declares only the inputs this resource manages: an operator may manage " +
					"just the operator inputs while the app team owns the user inputs, or vice versa. " +
					"Non-sensitive inputs show their plain value; sensitive inputs show only their hash. Set values in `spec.inputs`.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"value": schema.StringAttribute{
							MarkdownDescription: "Non-sensitive input value. This is a `jsonencode`d representation, for example `\"my-name\"` for a string or `16` for an integer.",
							Computed:            true,
						},
						"sensitive": secret.ReadOnlyResourceSchema(secret.ResourceSchemaOptions{
							MarkdownDescription: "Sensitive input value.",
						}),
						"value_type": schema.StringAttribute{
							MarkdownDescription: "Data type of the value. One of " + client.MeshBuildingBlockIOTypes.Markdown() + ".",
							Computed:            true,
						},
						"assignment_type": schema.StringAttribute{
							MarkdownDescription: "How the input value is assigned. Either " + client.MeshBuildingBlockInputAssignmentTypeUserInput.Markdown() +
								" or " +
								client.MeshBuildingBlockInputAssignmentTypePlatformOperatorManualInput.Markdown() + ".",
							Computed: true,
						},
					},
				},
				Computed:      true,
				PlanModifiers: []planmodifier.Map{mapplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

type buildingBlockModel struct {
	client.MeshBuildingBlockV2
	WaitForCompletion bool `tfsdk:"wait_for_completion"`
	PurgeOnDelete     bool `tfsdk:"purge_on_delete"`

	AllInputs map[string]buildingBlockAllInput `tfsdk:"all_inputs"`
	// Timeouts mirrors the schema's timeouts block so it round-trips through state via the generic
	// conversion layer (nil ↔ a null block). The effective durations are resolved separately with the
	// terraform-plugin-framework-timeouts helper in resolveTimeout.
	Timeouts *buildingBlockTimeouts `tfsdk:"timeouts"`
}

type buildingBlockTimeouts struct {
	Create *string `tfsdk:"create"`
	Update *string `tfsdk:"update"`
	Delete *string `tfsdk:"delete"`
}

type buildingBlockAllInput struct {
	Value          *string                                                 `tfsdk:"value"`
	Sensitive      *secret.HashOnly                                        `tfsdk:"sensitive"`
	ValueType      enum.Entry[client.MeshBuildingBlockIOType]              `tfsdk:"value_type"`
	AssignmentType enum.Entry[client.MeshBuildingBlockInputAssignmentType] `tfsdk:"assignment_type"`
}

// buildAllInput converts a single backend building block input into its read-only all_inputs
// representation: a sensitive input surfaces only its hash (never plaintext), a non-sensitive
// input its jsonencode'd value. Shared by the building_block resource and the
// building_blocks data source so both present an identical all_inputs shape.
func buildAllInput(in *client.MeshBuildingBlockInput, diags *diag.Diagnostics) (out buildingBlockAllInput) {
	if in.IsSensitive {
		// Guard against nil hash (backend returns null for unset sensitive inputs).
		if in.Value.X.Hash != nil {
			out.Sensitive = &secret.HashOnly{Hash: *in.Value.X.Hash}
		}
	} else {
		var err error
		out.Value, err = marshalAnyIfPresent(in.Value)
		if err != nil {
			// non-fatal here: record the diag and keep going.
			diags.AddError("Marshalling input value failed", err.Error())
		}
	}
	if in.ValueType != nil {
		out.ValueType = *in.ValueType
	}
	out.AssignmentType = in.AssignmentType
	return
}

func (m *buildingBlockModel) SetFromClientDto(dto *client.MeshBuildingBlockV2, isImport bool, diags *diag.Diagnostics) {
	// snapshot the configured inputs before the DTO overwrite, to decide which to keep in spec.inputs
	// versus surface only in all_inputs
	specInputs := maps.Clone(m.Spec.Inputs)
	// content_hash is TF-only (json:"-") and never returned by the backend, so preserve
	// the plan/state value across the DTO overwrite instead of letting it reset to null.
	contentHash := m.Spec.BuildingBlockDefinitionVersionRef.ContentHash
	m.MeshBuildingBlockV2 = *dto
	m.Spec.BuildingBlockDefinitionVersionRef.ContentHash = contentHash
	// kind is a fixed discriminator; force it so it round-trips even if the backend omits it.
	m.Spec.BuildingBlockDefinitionVersionRef.Kind = client.MeshObjectKind.BuildingBlockDefinitionVersion

	m.AllInputs = make(map[string]buildingBlockAllInput)

	mapToAllInput := func(in *client.MeshBuildingBlockInput) buildingBlockAllInput {
		return buildAllInput(in, diags)
	}

	for key, input := range m.Spec.Inputs {
		switch input.AssignmentType {
		case client.MeshBuildingBlockInputAssignmentTypeUserInput:
			isNullValue := !input.Value.HasX() && !input.Value.HasY() && !input.IsSensitive
			if _, exists := specInputs[key]; !exists {
				// This input is NOT declared by the current configuration. Surface it read-only in
				// all_inputs and drop it from spec.inputs — symmetric with platform-operator inputs —
				// so a configuration may manage only the inputs it declares without another party's
				// inputs showing as drift (e.g. a platform operator that sets only operator inputs while
				// the consumer owns the user inputs, or vice versa). The backend preserves inputs absent
				// from a PUT, so dropping them here is safe.
				//
				// EXCEPTION — import: there is no prior configuration, so a set (non-null) value must be
				// kept in spec.inputs to reflect what the imported block actually has; only phantom
				// null rows are dropped.
				if isImport && !isNullValue {
					m.AllInputs[key] = mapToAllInput(input)
					continue
				}
				delete(m.Spec.Inputs, key)
				m.AllInputs[key] = mapToAllInput(input)
				continue
			}
			// The key WAS declared. If the backend echoed a null value, keep the prior configured value.
			if isNullValue {
				if prior, ok := specInputs[key]; ok && (prior.Value.HasX() || prior.Value.HasY()) {
					m.Spec.Inputs[key] = prior
					m.AllInputs[key] = mapToAllInput(prior)
					continue
				}
			}
			m.AllInputs[key] = mapToAllInput(input)
		case client.MeshBuildingBlockInputAssignmentTypePlatformOperatorManualInput:
			if _, exists := specInputs[key]; !exists {
				// not declared by the configuration, so drop from spec.inputs and surface it read-only in all_inputs only
				delete(m.Spec.Inputs, key)
			}
			m.AllInputs[key] = mapToAllInput(input)
		default:
			// all other assignment types are not user-configurable
			delete(m.Spec.Inputs, key)
			m.AllInputs[key] = mapToAllInput(input)
		}
	}
}

func marshalAnyIfPresent(in clientTypes.SecretOrAny) (*string, error) {
	// JSON-encode so the value matches the jsontypes.Normalized attribute.
	if in.HasY() {
		marshalled, err := json.Marshal(in.Y)
		if err != nil {
			return nil, err
		}
		return new(string(marshalled)), nil
	}
	return nil, nil
}

var secretOrAnyValueToConverter = generic.WithValueToConverterFor[clientTypes.SecretOrAny](func(attributePath path.Path, in tftypes.Value) (out clientTypes.SecretOrAny, err error) {
	if in.IsKnown() && !in.IsNull() {
		var jsonValue string
		err = in.As(&jsonValue)
		if err != nil {
			return
		}
		err = json.Unmarshal([]byte(jsonValue), &out.Y)
	}
	return
})

var secretOrAnyValueFromConverter = generic.WithValueFromConverterFor[clientTypes.SecretOrAny](generic.ValueFromConverterForTypedNilHandler[string](),
	func(_ path.Path, in clientTypes.SecretOrAny) (tftypes.Value, error) {
		marshalled, err := marshalAnyIfPresent(in)
		if err != nil {
			return tftypes.Value{}, err
		}
		return generic.ValueFrom(marshalled)
	})

func buildingBlockConverterOptions(ctx context.Context, config, plan, state generic.AttributeGetter) generic.ConverterOptions {
	type buildingBlockInputWithSensitive struct {
		client.MeshBuildingBlockInput
		Sensitive *secret.Secret `tfsdk:"sensitive"`
	}

	return generic.ConverterOptions{
		// clientTypes.Any output value — needed both directions (Set to state, Get back on refresh).
		withValueFromConverterForClientTypeAny(),
		withValueToConverterForClientTypeAny(),

		generic.WithSliceTypeAsSet(clientTypes.IsSet),

		// inputs: from Client DTO to model
		generic.WithValueFromConverterFor[client.MeshBuildingBlockInput](
			func() (tftypes.Value, error) {
				return generic.ValueFrom[*buildingBlockInputWithSensitive](nil, secretOrAnyValueFromConverter)
			},
			func(attributePath path.Path, in client.MeshBuildingBlockInput) (tftypes.Value, error) {
				// Note that client.MeshBuildingBlockInput.UnmarshalJSON ensures that the IsSensitive flag is consistent with the clientTypes.SecretOrAny aka Variant[X, Y] state
				out := buildingBlockInputWithSensitive{MeshBuildingBlockInput: in}
				if in.IsSensitive {
					secretValue, err := secret.ValueFromConverter(ctx, plan, state, attributePath.AtName("sensitive"), in.Value.X)
					if err != nil {
						return tftypes.Value{}, err
					}
					out.Sensitive, err = generic.ValueTo[*secret.Secret](secretValue)
					if err != nil {
						return tftypes.Value{}, err
					}
				}
				return generic.ValueFrom(out,
					generic.WithAttributePath(attributePath),
					secretOrAnyValueFromConverter,
				)
			}),

		// inputs: from model to Client DTO
		generic.WithValueToConverterFor[client.MeshBuildingBlockInput](func(attributePath path.Path, in tftypes.Value) (client.MeshBuildingBlockInput, error) {
			model, err := generic.ValueTo[buildingBlockInputWithSensitive](in, secretOrAnyValueToConverter, generic.WithSetUnknownValueToZero())
			if err != nil {
				return client.MeshBuildingBlockInput{}, err
			}
			if model.Sensitive != nil {
				model.IsSensitive = true
				model.Value.X, err = secret.ValueToConverter(ctx, config, plan, state, attributePath.AtName("sensitive"))
				if err != nil {
					return client.MeshBuildingBlockInput{}, err
				}
			}
			return model.MeshBuildingBlockInput, nil
		}),
	}
}

// resolveTimeout reads the configured timeout for the given operation ("create"/"update"/"delete")
// from the resource's timeouts block, falling back to defaultBuildingBlockTimeout when unset. The
// getter is the plan for create/update and the state for delete.
func resolveTimeout(ctx context.Context, getter generic.AttributeGetter, op string, diags *diag.Diagnostics) time.Duration {
	var configured timeouts.Value
	diags.Append(getter.GetAttribute(ctx, path.Root("timeouts"), &configured)...)

	var (
		timeout time.Duration
		tdiags  diag.Diagnostics
	)
	switch op {
	case "create":
		timeout, tdiags = configured.Create(ctx, defaultBuildingBlockTimeout)
	case "update":
		timeout, tdiags = configured.Update(ctx, defaultBuildingBlockTimeout)
	case "delete":
		timeout, tdiags = configured.Delete(ctx, defaultBuildingBlockTimeout)
	}
	diags.Append(tdiags...)
	return timeout
}

func (r *buildingBlockResource) addRunFailureDiagnostics(
	ctx context.Context,
	diags *diag.Diagnostics,
	summary string,
	pollErr error,
	bb *client.MeshBuildingBlockV2,
) {
	if pollErr != nil {
		diags.AddError(summary, pollErr.Error())
	}
	if bb == nil || bb.Status == nil || bb.Status.LatestRunUuid == nil {
		return
	}
	logs, err := r.BuildingBlockRunClient.GetLogs(ctx, *bb.Status.LatestRunUuid)
	if err != nil {
		// Fetching run logs can legitimately fail: the building block definition may have run
		// transparency disabled, or the caller's permissions may not allow reading them. Surface it
		// as a warning so the run failure itself (the error added above) still stands on its own.
		diags.AddWarning(
			"Could not fetch run logs",
			"The building block run failed but its logs could not be retrieved. This can happen when the "+
				"building block definition has run transparency disabled or your permissions do not allow "+
				"reading run logs. Inspect the run in meshPanel for details. Underlying error: "+err.Error(),
		)
		return
	}

	// truncate cuts s to at most limit runes (not bytes), slicing by runes to avoid splitting
	// multi-byte UTF-8 sequences; it appends "…" when truncated.
	truncate := func(s string, limit int) string {
		count := 0
		for i := range s {
			if count == limit {
				return s[:i] + "…"
			}
			count++
		}
		return s
	}

	// Report only the first failed step: subsequent steps usually fail as a cascade of the first, so
	// listing them all just adds noise. Include both the user and system messages so the error
	// carries everything the API returned for that step. This is surfaced as an error (not a warning)
	// because we only reach here when the run failed and the apply is already erroring: the step log
	// is the actionable detail behind that failure, so it belongs with the error rather than alongside
	// it. It only appears when the run logs are readable (run transparency on / sufficient permissions);
	// otherwise the unreadable-logs warning above stands and the run-failure error carries on its own.
	for _, step := range logs.Steps {
		if step.Status != string(client.BuildingBlockStatusFailed) {
			continue
		}
		detail := fmt.Sprintf("Step %q is in status %s.", step.DisplayName, step.Status)
		if step.UserMessage != nil {
			detail += "\nMessage: " + truncate(*step.UserMessage, 2000)
		}
		if step.SystemMessage != nil {
			detail += "\nSystem message: " + truncate(*step.SystemMessage, 2000)
		}
		diags.AddError("Run step failed: "+step.DisplayName, detail)
		break
	}
}

// addWaitingForInputWarning emits the "waiting for input" warning for a building block that has
// short-circuited to a terminal-but-waiting state. Shared by awaitRun and Create's short-circuit.
func addWaitingForInputWarning(diags *diag.Diagnostics, bb *client.MeshBuildingBlockV2) {
	uuid := "unknown"
	if bb.Metadata.Uuid != nil {
		uuid = *bb.Metadata.Uuid
	}
	diags.AddWarning(
		"Building block run is waiting for input or approval",
		fmt.Sprintf("Building block %s is in status %s. Resolve the pending input or approval in meshPanel to complete the run; outputs are not yet available.", uuid, bb.Status.Status),
	)
}

// awaitRun polls until the building block reaches a terminal state (when waitForCompletion is set).
//
// A run triggered by the preceding create/update is reflected immediately as a PENDING status: the backend
// eager-sets PENDING whenever a run will follow (a forced run, a version upgrade, or supplying the inputs
// that make a parked block runnable), so the provider no longer has to disambiguate a stale previous-run
// status from a freshly-triggered one. We simply poll the status: PENDING/IN_PROGRESS keep polling,
// SUCCEEDED/FAILED/ABORTED are terminal, and a WAITING_FOR_*_INPUT status means the block is parked and
// cannot proceed from this apply (a runnable block would be PENDING) — surfaced as a non-fatal warning
// rather than polling to the timeout.
func (r *buildingBlockResource) awaitRun(
	ctx context.Context,
	diags *diag.Diagnostics,
	uuid string,
	waitForCompletion bool,
	timeout time.Duration,
) *client.MeshBuildingBlockV2 {
	if !waitForCompletion {
		return nil
	}
	predicate := func(bb *client.MeshBuildingBlockV2) (bool, error) {
		if bb == nil {
			// The block 404'd while we were waiting (purged, or its definition deleted out-of-band). Stop
			// polling with a clear error instead of dereferencing a nil block in the checks below.
			return false, fmt.Errorf("building block disappeared while waiting for its run to complete")
		}
		if bb.Status != nil && bb.IsWaitingForInput() {
			// Parked waiting for input this apply cannot supply — terminal-but-non-fatal (warned below).
			return true, nil
		}
		return bb.CreateSuccessful()
	}
	var final *client.MeshBuildingBlockV2
	err := poll.AtMostFor(timeout, r.BuildingBlockClient.ReadFunc(uuid),
		poll.WithLastResultTo(&final)).
		Until(ctx, predicate)
	if err != nil {
		r.addRunFailureDiagnostics(ctx, diags, "Building block run failed", err, final)
	} else if final != nil && final.IsWaitingForInput() {
		addWaitingForInputWarning(diags, final)
	}
	return final
}

func (r *buildingBlockResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	converterOptions := buildingBlockConverterOptions(ctx, req.Config, req.Plan, nil)

	plan := generic.Get[buildingBlockModel](ctx, req.Plan, &resp.Diagnostics, converterOptions.Append(generic.WithSetUnknownValueToZero())...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Send only Spec — Metadata.Uuid and Status are assigned by the backend.
	created, err := r.BuildingBlockClient.Create(ctx, &client.MeshBuildingBlockV2{Spec: plan.Spec})
	if err != nil {
		resp.Diagnostics.AddError("Error creating building block", err.Error())
		return
	}
	plan.SetFromClientDto(created, false, &resp.Diagnostics)
	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, plan, converterOptions...)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Short-circuit the poll if the POST response already satisfies CreateSuccessful.
	if plan.WaitForCompletion {
		if done, _ := created.CreateSuccessful(); done {
			// already terminal — no need to poll. If it short-circuited because the block parked in a
			// WAITING_* state, surface the same warning awaitRun would have, so the user isn't shown a clean
			// apply on a stuck block.
			if created.IsWaitingForInput() {
				addWaitingForInputWarning(&resp.Diagnostics, created)
			}
			return
		}
		timeout := resolveTimeout(ctx, req.Plan, "create", &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		final := r.awaitRun(ctx, &resp.Diagnostics, *created.Metadata.Uuid, true, timeout)
		if final != nil {
			plan.SetFromClientDto(final, false, &resp.Diagnostics)
			resp.Diagnostics.Append(generic.Set(ctx, &resp.State, plan, converterOptions...)...)
		}
	}
}

func (r *buildingBlockResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	converterOptions := buildingBlockConverterOptions(ctx, nil, nil, req.State)
	state := generic.Get[buildingBlockModel](ctx, req.State, &resp.Diagnostics, converterOptions.Append(generic.WithSetUnknownValueToZero())...)
	if resp.Diagnostics.HasError() {
		return
	}
	readDto, err := r.BuildingBlockClient.Read(ctx, *state.Metadata.Uuid)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read building block", fmt.Sprintf("Reading building block '%s' failed: %s", *state.Metadata.Uuid, err.Error()))
		return
	}
	// The block is gone when the read 404s (nil, e.g. after a hard delete/purge) or when the backend
	// returns it soft-deleted (lifecycle DELETED — a soft delete does not 404). Either way, drop it.
	if readDto == nil || (readDto.Status != nil && readDto.Status.Lifecycle.State == client.BuildingBlockLifecycleStateDeleted) {
		resp.State.RemoveResource(ctx)
		return
	}

	// Preserve config-only flags from prior state; on import apply schema defaults.
	waitForCompletion := generic.GetAttribute[*bool](ctx, req.State, path.Root("wait_for_completion"), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	var waitForCompletionBool bool
	if waitForCompletion == nil {
		// Import context: no prior state value, apply schema default (true).
		waitForCompletionBool = true
	} else {
		waitForCompletionBool = *waitForCompletion
	}
	purgeOnDelete := state.PurgeOnDelete

	// waitForCompletion == nil signals an import (no prior state); see the SetFromClientDto USER_INPUT
	// handling, which keeps set user inputs on import but drops un-declared ones on a normal refresh.
	state.SetFromClientDto(readDto, waitForCompletion == nil, &resp.Diagnostics)
	state.WaitForCompletion = waitForCompletionBool
	state.PurgeOnDelete = purgeOnDelete
	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, state, converterOptions...)...)
}

// requiresReplaceParentsWhenVersionUnchanged forces replacement when parent_building_blocks changes
// but the building block definition version does NOT. The backend only accepts a parent change as part
// of a version-change PUT (requireParentsUnchanged on a same-version PUT → 400), so an in-place update of
// parents alone is doomed. When the version uuid also changes, the in-place upgrade carries the parent
// change, so no replacement is needed. RequiresReplaceIf is only invoked when the attribute actually changed.
func requiresReplaceParentsWhenVersionUnchanged(ctx context.Context, req planmodifier.SetRequest, resp *setplanmodifier.RequiresReplaceIfFuncResponse) {
	// Not applicable on create (no prior state) or destroy (no plan).
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	versionPath := path.Root("spec").AtName("building_block_definition_version_ref").AtName("uuid")
	var planVersion, stateVersion types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, versionPath, &planVersion)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, versionPath, &stateVersion)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the version is unknown (e.g. wired to another resource), assume an upgrade is in play and let the
	// in-place path handle the parents. Only replace when the version is provably unchanged.
	if planVersion.IsUnknown() {
		return
	}
	resp.RequiresReplace = planVersion.Equal(stateVersion)
}

// rerunNeeded is the single shared predicate determining whether an Update should trigger a run.
// Used by both ModifyPlan and Update.
// Semantics:
//  1. version-ref uuid differs → rerun
//  2. content_hash:
//     - rerun when plan non-nil and state nil (newly set) → rerun.
//     - both non-nil and hashes are different → rerun.
//     - plan nil + state non-nil (user removed content_hash) → NOT a rerun.
//     - an arbitrary (non-versioned) content_hash value the user sets to force a manual rerun → rerun.
//  3. inputs differ → rerun.
//  4. parent set differs → rerun.
func rerunNeeded(plan, state client.MeshBuildingBlockV2Spec) bool {
	// uuid change detection
	if plan.BuildingBlockDefinitionVersionRef.Uuid != state.BuildingBlockDefinitionVersionRef.Uuid {
		return true
	}

	// content_hash change detection
	planHash := plan.BuildingBlockDefinitionVersionRef.ContentHash
	stateHash := state.BuildingBlockDefinitionVersionRef.ContentHash
	if planHash != nil {
		if stateHash == nil || compareContentHashes(*planHash, *stateHash) == hashDifferent {
			return true
		}
	}

	// inputs change detection
	if planInputsChanged(plan.Inputs, state.Inputs) {
		return true
	}

	// parent building blocks change detection
	if !reflect.DeepEqual(plan.ParentBuildingBlocks, state.ParentBuildingBlocks) {
		return true
	}

	return false
}

func compareContentHashes(planHash, stateHash string) hashComparison {
	planParsed, err := getVersionedHashFromString(planHash)
	if err != nil {
		return stringBasedHashComparison(planHash, stateHash)
	}

	cmp := planParsed.compareToStored(stateHash)
	// hashIncomparable means either a different algorithm version, or the state string is free-form.
	// Only the latter should fall back to a plain comparison; a genuine version mismatch stays incomparable.
	if cmp == hashIncomparable {
		if _, err := getVersionedHashFromString(stateHash); err != nil {
			return stringBasedHashComparison(planHash, stateHash) // state side free-form → plain comparison
		}
	}
	return cmp
}

func stringBasedHashComparison(planHash, stateHash string) hashComparison {
	if planHash != stateHash {
		return hashDifferent
	}
	return hashSame
}

func (r *buildingBlockResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return // destroy — nothing to modify
	}

	secret.WalkSecretPathsIn(req.Plan.Raw, &resp.Diagnostics, func(attributePath path.Path, diags *diag.Diagnostics) {
		secret.SetToUnknownIfVersionChangedOrCreated(ctx, req.Plan, req.State, &resp.Plan)(attributePath, diags)
	})

	if req.State.Raw.IsNull() {
		return // create — no prior state to diff against
	}

	// Past the guards above only Update reaches here.
	converterOptions := buildingBlockConverterOptions(ctx, req.Config, req.Plan, req.State)

	// Mark the run-derived read-only attributes (status, outputs, all_inputs) unknown so the apply can
	// resolve them after the triggered run.
	triggerRun := func() {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("status").AtName("status"), types.StringUnknown())...)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("status").AtName("latest_run_uuid"), types.StringUnknown())...)
		// latest_dry_run_uuid resolves to null once the applying run is the newest; mark it unknown so the
		// apply can resolve it without a "provider produced inconsistent result" error if state carried a
		// non-null dry uuid (e.g. an out-of-band dry run was previously the newest).
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("status").AtName("latest_dry_run_uuid"), types.StringUnknown())...)

		var outputs types.Map
		resp.Diagnostics.Append(resp.Plan.GetAttribute(ctx, path.Root("status").AtName("outputs"), &outputs)...)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("status").AtName("outputs"), types.MapUnknown(outputs.ElementType(ctx)))...)

		var allInputs types.Map
		resp.Diagnostics.Append(resp.Plan.GetAttribute(ctx, path.Root("all_inputs"), &allInputs)...)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("all_inputs"), types.MapUnknown(allInputs.ElementType(ctx)))...)
	}

	// Check for unknown rerun-relevant fields BEFORE converting spec.
	// If version uuid or content_hash is unknown (wired to another resource being replaced),
	// conservatively trigger run and return — do NOT try to convert-and-error.
	var planVersionUuid types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("building_block_definition_version_ref").AtName("uuid"), &planVersionUuid)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if planVersionUuid.IsUnknown() {
		triggerRun()
		return
	}

	var planContentHash types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("building_block_definition_version_ref").AtName("content_hash"), &planContentHash)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if planContentHash.IsUnknown() {
		triggerRun()
		return
	}

	stateSpec := generic.GetAttribute[client.MeshBuildingBlockV2Spec](ctx, req.State, path.Root("spec"), &resp.Diagnostics, converterOptions...)
	planSpec := generic.GetAttribute[client.MeshBuildingBlockV2Spec](ctx, req.Plan, path.Root("spec"), &resp.Diagnostics, converterOptions...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A sensitive-input rotation (changed secret_version) is invisible to rerunNeeded because the
	// converted spec collapses secrets to {plaintext|hash}; detect it directly and OR it in, using the same
	// helper Update uses so plan and apply agree.
	secretRotated := secret.AnyVersionChangedIn(ctx, req.Plan.Raw, req.Plan, req.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// When plan and state content_hash differ only by their hash-algorithm version (see
	// classifyContentHash), the building block is deliberately NOT re-run. Warn so the user knows a
	// genuine content change - if any - was not picked up, and how to force a re-run.
	if planHash := planSpec.BuildingBlockDefinitionVersionRef.ContentHash; planHash != nil {
		if stateHash := stateSpec.BuildingBlockDefinitionVersionRef.ContentHash; stateHash != nil {
			if compareContentHashes(*planHash, *stateHash) == hashIncomparable {
				resp.Diagnostics.AddAttributeWarning(
					path.Root("spec").AtName("building_block_definition_version_ref").AtName("content_hash"),
					"Building block definition version content hash version changed; not re-running",
					"The referenced version's content_hash changed only because it was produced by a different "+
						"hash-algorithm version (for example after upgrading the provider), so the building block was "+
						"not re-run. If the version's content actually changed and you want to force a re-run, set "+
						"spec.building_block_definition_version_ref.content_hash to an arbitrary new value.",
				)
			}
		}
	}

	if rerunNeeded(planSpec, stateSpec) || secretRotated {
		triggerRun()
	}
}

// planInputsChanged reports whether the inputs DECLARED in the plan differ from state — an input added
// to the plan, or one whose value changed. Inputs present in state but ABSENT from the plan are ignored:
// the backend preserves inputs omitted from a PUT, so dropping an input from configuration is not a
// provisioning change and must not make the provider await a run that never starts (it also lets one
// configuration manage a subset of inputs while another party owns the rest). It uses reflect.DeepEqual
// per-input because MeshBuildingBlockInput holds an `any` value (clientTypes.SecretOrAny) that may wrap a
// non-comparable type decoded from JSON; comparing those with == panics at runtime.
func planInputsChanged(plan, state map[string]*client.MeshBuildingBlockInput) bool {
	for key, planInput := range plan {
		stateInput, ok := state[key]
		if !ok || !reflect.DeepEqual(planInput, stateInput) {
			return true
		}
	}
	return false
}

func (r *buildingBlockResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	converterOptions := buildingBlockConverterOptions(ctx, req.Config, req.Plan, req.State)
	plan := generic.Get[buildingBlockModel](ctx, req.Plan, &resp.Diagnostics, converterOptions.Append(generic.WithSetUnknownValueToZero())...)
	state := generic.Get[buildingBlockModel](ctx, req.State, &resp.Diagnostics, converterOptions.Append(generic.WithSetUnknownValueToZero())...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Compute rerun condition BEFORE calling the client.
	// Also rerun when a sensitive input was rotated (secret_version changed) — invisible to
	// rerunNeeded since the converted spec hides it. Same helper as ModifyPlan, so plan and apply agree.
	secretRotated := secret.AnyVersionChangedIn(ctx, req.Plan.Raw, req.Plan, req.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		// AnyVersionChangedIn can append diagnostics; bail before the side-effecting PUT below so we never
		// mutate the backend on an unresolved/invalid secret state (and lose the rotation signal).
		return
	}
	needsRun := rerunNeeded(plan.Spec, state.Spec) || secretRotated

	// A version-change PUT is only accepted by the backend when the block is in a completed state
	// (SUCCEEDED, FAILED, ABORTED); otherwise the backend 409s. Pre-check and fail fast with a clear message
	// rather than letting a raw 409 reach the user.
	versionChanging := plan.Spec.BuildingBlockDefinitionVersionRef.Uuid != state.Spec.BuildingBlockDefinitionVersionRef.Uuid
	if versionChanging && state.Status != nil {
		switch state.Status.Status {
		case client.BuildingBlockStatusSucceeded, client.BuildingBlockStatusFailed, client.BuildingBlockStatusAborted:
			// completed — upgrade is allowed
		default:
			resp.Diagnostics.AddError(
				"Building block must be in a completed state to change its definition version",
				fmt.Sprintf("Changing the building block definition version requires status SUCCEEDED, FAILED, or ABORTED; current status is %s. Resolve any pending input or run first, then retry the upgrade.", state.Status.Status),
			)
			return
		}
	}

	// Send only Metadata+Spec — Status is read-only and must not be passed to PUT.
	updated, err := r.BuildingBlockClient.Update(ctx, &client.MeshBuildingBlockV2{
		Metadata: plan.Metadata,
		Spec:     plan.Spec,
	})
	if err != nil {
		// Map a backend 409 (e.g. a version change on a non-completed block that slipped past the
		// pre-check, or a concurrent run) to a clear, actionable message instead of a raw HTTP error.
		if httpErr, ok := errors.AsType[client.HttpError](err); ok && httpErr.IsConflict() {
			resp.Diagnostics.AddError(
				"Building block cannot be updated in its current state",
				fmt.Sprintf("The backend rejected the update with a conflict (409). This usually means the building block is not in a completed state for the requested change (for example a version upgrade while a run is still pending). Resolve any pending input or run first, then retry. Underlying error: %s", err.Error()),
			)
			return
		}
		resp.Diagnostics.AddError("Error updating building block", err.Error())
		return
	}

	// The backend PUT triggers an apply run on a backend-visible change (version upgrade or an actual
	// input/parent change; secret rotation changes the stored value). content_hash is provider-only (never
	// sent to the backend), so a content_hash-only rerun is the one case the PUT can't cover — force it via
	// an explicit (non-dry) trigger-run.
	backendWillRun := versionChanging ||
		planInputsChanged(plan.Spec.Inputs, state.Spec.Inputs) ||
		!reflect.DeepEqual(plan.Spec.ParentBuildingBlocks, state.Spec.ParentBuildingBlocks) ||
		secretRotated
	if needsRun && !backendWillRun {
		if err := r.BuildingBlockClient.TriggerRun(ctx, *updated.Metadata.Uuid); err != nil {
			resp.Diagnostics.AddError("Error triggering building block run", err.Error())
			return
		}
	}

	// Wait for the triggered run (whether the PUT or the explicit trigger-run started it).
	effective := updated
	if needsRun {
		timeout := resolveTimeout(ctx, req.Plan, "update", &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		final := r.awaitRun(ctx, &resp.Diagnostics, *updated.Metadata.Uuid, plan.WaitForCompletion, timeout)
		if final != nil {
			effective = final
		}
	}

	plan.SetFromClientDto(effective, false, &resp.Diagnostics)
	// Reuse the converter options built at the top of Update (same config/plan/state getters), so
	// secret_version resolves consistently with Read without rebuilding the converters.
	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, plan, converterOptions...)...)
}

func (r *buildingBlockResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	uuid := generic.GetAttribute[string](ctx, req.State, path.Root("metadata").AtName("uuid"), &resp.Diagnostics)
	purgeOnDelete := generic.GetAttribute[bool](ctx, req.State, path.Root("purge_on_delete"), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.BuildingBlockClient.Delete(ctx, uuid, purgeOnDelete); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting building block",
			"Could not delete building block, unexpected error: "+err.Error(),
		)
		return
	}
	// Always poll until the block is gone — lifecycle DELETED for a soft delete, or a 404 for a purge —
	// regardless of wait_for_completion. A plain DELETE schedules an async destroy run (returns 202) and
	// only flips to DELETED once the cloud resources are torn down; a PURGE skips the run and reaches
	// DELETED/404 essentially immediately. Returning early on the 202 would let the resource be dropped
	// from state while teardown is still in flight, racing any dependent delete that follows — e.g.
	// deleting the building block definition then fails 409 "existing BuildingBlocks referencing it".
	timeout := resolveTimeout(ctx, req.State, "delete", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	var final *client.MeshBuildingBlockV2
	err := poll.AtMostFor(timeout, r.BuildingBlockClient.ReadFunc(uuid),
		poll.WithLastResultTo(&final)).
		Until(ctx, (*client.MeshBuildingBlockV2).DeletionSuccessful)
	if err != nil {
		r.addRunFailureDiagnostics(ctx, &resp.Diagnostics, "Building block deletion failed", err, final)
		return // keep resource in state on failure
	}
}

func (r *buildingBlockResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("metadata").AtName("uuid"), req, resp)
	// content_hash is json:"-" and never returned by the API, so import leaves it null.
	// With the rerun semantics (state nil + plan non-nil = "newly set" → rerun), this is by design:
	// the first apply after import triggers a run if content_hash is set in config.
}
