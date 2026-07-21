package provider

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/secret"
)

var (
	_ resource.Resource                   = &buildingBlockDefinitionResource{}
	_ resource.ResourceWithConfigure      = &buildingBlockDefinitionResource{}
	_ resource.ResourceWithImportState    = &buildingBlockDefinitionResource{}
	_ resource.ResourceWithModifyPlan     = &buildingBlockDefinitionResource{}
	_ resource.ResourceWithValidateConfig = &buildingBlockDefinitionResource{}
)

func NewBuildingBlockDefinitionResource() resource.Resource {
	return &buildingBlockDefinitionResource{}
}

type buildingBlockDefinitionResource struct {
	buildingBlockDefinitionClient        client.MeshBuildingBlockDefinitionClient
	buildingBlockDefinitionVersionClient client.MeshBuildingBlockDefinitionVersionClient
}

func (r *buildingBlockDefinitionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_building_block_definition"
}

func (r *buildingBlockDefinitionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		r.buildingBlockDefinitionClient = client.BuildingBlockDefinition
		r.buildingBlockDefinitionVersionClient = client.BuildingBlockDefinitionVersion
	})...)
}

func (r *buildingBlockDefinitionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	converterOptions := buildingBlockDefinitionConverterOptions().
		Append(buildingBlockDefinitionVersionConverterOptions(ctx, req.Config, req.Plan, nil)...)
	plan := generic.Get[buildingBlockDefinition](ctx, req.Plan, &resp.Diagnostics, converterOptions.Append(generic.WithSetUnknownValueToZero())...)
	if resp.Diagnostics.HasError() {
		return
	}
	createdDto, err := r.buildingBlockDefinitionClient.Create(ctx, client.MeshBuildingBlockDefinition{
		Metadata: plan.Metadata,
		Spec:     plan.Spec,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating MeshBuildingBlockDefinition", err.Error())
		return
	}

	// Set spec/metadata of BBD immediately after successful creation. Keep the tags the user declared
	// rather than the superset the API returns (an entry for every defined tag property plus injected
	// restricted-tag defaults), which would break plan/apply consistency. SetFromClientDto overwrites
	// Metadata wholesale, so capture the planned tags first and restore them afterwards.
	plannedTags := plan.Metadata.Tags
	plan.SetFromClientDto(createdDto, &resp.Diagnostics)
	plan.Metadata.Tags = plannedTags
	resp.Diagnostics.Append(generic.SetAttributeTo(ctx, &resp.State, path.Root("metadata"), plan.Metadata, buildingBlockDefinitionConverterOptions()...)...)
	resp.Diagnostics.Append(generic.SetAttributeTo(ctx, &resp.State, path.Root("spec"), plan.Spec, buildingBlockDefinitionConverterOptions()...)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bbdUuid := *createdDto.Metadata.Uuid
	versions, err := r.buildingBlockDefinitionVersionClient.List(ctx, bbdUuid)
	if err != nil {
		resp.Diagnostics.AddError("Error listing versions after creating definition", err.Error())
		return
	} else if len(versions) != 1 {
		resp.Diagnostics.AddError("Created version not found", fmt.Sprintf(
			"Expected 1 version (empty hull), but got %d for newly created building block '%s', ID=%s",
			len(versions), createdDto.Spec.DisplayName, bbdUuid,
		))
	}
	createdEmptyVersion := versions[0]

	if resp.Diagnostics.HasError() {
		return
	}

	// Updating the empty created version with provided configuration to complete creation
	versionUuid := createdEmptyVersion.Metadata.Uuid
	createVersionSpecDto := versionSpecDtoFromPlan(ctx, plan, bbdUuid, req.Config, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	createdVersionDto, err := r.buildingBlockDefinitionVersionClient.Update(ctx, versionUuid, createdDto.Metadata.OwnedByWorkspace, createVersionSpecDto)
	if err != nil {
		resp.Diagnostics.AddError("Error updating initial version", fmt.Sprintf(
			"Building block '%s', uuid=%s was just created, and the initial version '%s' failed to update with given version_spec configuration. "+
				"Most likely schema validation is insufficient and the API received an invalid or incomplete JSON payload.\n"+
				"Error: %s",
			createdDto.Spec.DisplayName, bbdUuid, versionUuid, err.Error(),
		))
		return
	}

	plan.SetFromVersionClientDtos(&resp.Diagnostics, generic.KnownValue(plan.VersionSpec.Draft), bbdUuid, *createdVersionDto)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, plan, converterOptions...)...)
}

func (r *buildingBlockDefinitionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	stateMetadata := generic.GetAttribute[client.MeshBuildingBlockDefinitionMetadata](ctx, req.State, path.Root("metadata"), &resp.Diagnostics, generic.WithSetUnknownValueToZero())
	if resp.Diagnostics.HasError() {
		return
	}
	bbdUuid := *stateMetadata.Uuid

	definitionDto, err := r.buildingBlockDefinitionClient.Read(ctx, bbdUuid)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get building block definition", fmt.Sprintf("Reading the existing BBD '%s' failed: %s", bbdUuid, err.Error()))
		return
	} else if definitionDto == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	state := buildingBlockDefinition{
		Metadata: definitionDto.Metadata,
		Spec:     definitionDto.Spec,
	}

	// Keep only the tags we already track. The API returns a superset (every schema property plus
	// injected restricted-tag defaults) that the caller may be unable to manage, so mirroring it
	// verbatim would surface as drift. On import there is no prior state (tags is null); we keep the
	// full set so a normal import round-trips.
	state.Metadata.Tags = reconcileTrackedTags(ctx, req.State, path.Root("metadata").AtName("tags"), state.Metadata.Tags, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	versionDtos, err := r.buildingBlockDefinitionVersionClient.List(ctx, bbdUuid)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list versions for definition", fmt.Sprintf("Cannot list version for BBD %s: %s", bbdUuid, err.Error()))
		return
	} else if len(versionDtos) == 0 {
		// TODO maybe consider removing the inconsistent BBD resource again from the state?
		resp.Diagnostics.AddError("No BBD versions found", fmt.Sprintf(
			"Expected at least one version, but got none for building block '%s', ID=%s",
			definitionDto.Spec.DisplayName, bbdUuid,
		))
	}
	// Refresh reflects the actual latest-version state: derive draft from it (as the definitions data
	// source does) so an external switch to DRAFT is noticed instead of a stale draft=false persisting.
	state.SetFromVersionClientDtos(&resp.Diagnostics, deriveDraftFromLatestVersion(versionDtos), bbdUuid, versionDtos...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, state,
		buildingBlockDefinitionConverterOptions().Append(buildingBlockDefinitionVersionConverterOptions(ctx, nil, nil, req.State)...)...)...)
}

// outputStringAttr safely extracts a string attribute from a nested output object's attribute map,
// returning a null string when the key is absent or not a string (never panics on unexpected shapes).
func outputStringAttr(attrs map[string]attr.Value, name string) types.String {
	if s, ok := attrs[name].(types.String); ok {
		return s
	}
	return types.StringNull()
}

func (r *buildingBlockDefinitionResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	manualPath := path.Root("version_spec").AtName("implementation").AtName("manual")
	inputsPath := path.Root("version_spec").AtName("inputs")
	outputsPath := path.Root("version_spec").AtName("outputs")

	var manual types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, manualPath, &manual)...)
	var inputs, outputs types.Map
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, inputsPath, &inputs)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, outputsPath, &outputs)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Nothing declared -> nothing to validate (the outputs schema is optional+computed).
	if outputs.IsNull() || outputs.IsUnknown() {
		return
	}
	outputElems := outputs.Elements()

	// The outputs object is shared by manual and non-manual definitions; the schema had to loosen
	// display_name/type to optional+computed to accommodate manual's sparse-override model, so the
	// per-implementation contract is re-imposed here (the framework cannot flip Required per implementation).
	if manual.IsNull() || manual.IsUnknown() {
		// Non-manual: outputs are configured explicitly, so type and display_name remain mandatory.
		for key, elem := range outputElems {
			obj, ok := elem.(types.Object)
			if !ok || obj.IsNull() || obj.IsUnknown() {
				continue
			}
			attrs := obj.Attributes()
			if outputStringAttr(attrs, "type").IsNull() {
				resp.Diagnostics.AddAttributeError(outputsPath.AtMapKey(key),
					"output type is required",
					"For non-manual building block definitions, every declared output must set 'type'.")
			}
			if outputStringAttr(attrs, "display_name").IsNull() {
				resp.Diagnostics.AddAttributeError(outputsPath.AtMapKey(key),
					"output display_name is required",
					"For non-manual building block definitions, every declared output must set 'display_name'.")
			}
		}
		return
	}

	// Manual: outputs are a sparse override. The backend derives type and positional display_order and merges
	// the sent subset; the provider prunes the response back to the tracked subset with a diff rule
	// (assignment_type != NONE OR display_name != the input's). For that rule to be lossless - a hard
	// requirement for stable per-version content hashes - every accepted override must be reconstructable.
	// So: type is always derived (reject it); a no-op override (nothing but display_order, or a display_name
	// equal to the input's, with NONE assignment) cannot be reconstructed (reject it). Checks that need a value
	// which is unknown at plan are skipped (best-effort); the invariant is guaranteed for fully-known config.
	none := client.MeshBuildingBlockDefinitionOutputAssignmentTypeNone.String()
	inputsKnown := !inputs.IsNull() && !inputs.IsUnknown()
	var inputElems map[string]attr.Value
	if inputsKnown {
		inputElems = inputs.Elements()
	}
	for key, elem := range outputElems {
		obj, ok := elem.(types.Object)
		if !ok || obj.IsNull() || obj.IsUnknown() {
			continue
		}
		attrs := obj.Attributes()

		if t := attrs["type"]; t != nil && !t.IsNull() {
			resp.Diagnostics.AddAttributeError(outputsPath.AtMapKey(key).AtName("type"),
				"output type must not be set on manual building blocks",
				"For manual building block definitions the output type is always derived from the matching input. Remove 'type' from this output.")
		}

		if inputsKnown {
			if _, ok := inputElems[key]; !ok {
				resp.Diagnostics.AddAttributeError(outputsPath.AtMapKey(key),
					"declared output has no matching input",
					fmt.Sprintf("Manual building block outputs are keyed by their matching input; output %q has no matching input. "+
						"Remove it, or add an input with the same key.", key))
				continue
			}
		}

		assignment := outputStringAttr(attrs, "assignment_type")
		displayName := outputStringAttr(attrs, "display_name")

		assignmentProvablyNone := assignment.IsNull() || (!assignment.IsUnknown() && assignment.ValueString() == none)
		displayNameProvablyNotOverride := displayName.IsNull()
		if !displayName.IsNull() && !displayName.IsUnknown() && inputsKnown {
			if inputObj, ok := inputElems[key].(types.Object); ok {
				inputDisplayName := outputStringAttr(inputObj.Attributes(), "display_name")
				if !inputDisplayName.IsNull() && !inputDisplayName.IsUnknown() && displayName.ValueString() == inputDisplayName.ValueString() {
					displayNameProvablyNotOverride = true
				}
			}
		}
		if assignmentProvablyNone && displayNameProvablyNotOverride {
			resp.Diagnostics.AddAttributeError(outputsPath.AtMapKey(key),
				"manual building block output override has no effect",
				fmt.Sprintf("Output %q does not override anything: set assignment_type to a value other than %s, or a display_name "+
					"that differs from the matching input's. Overriding display_order alone is not supported.", key, none))
		}
	}
}

// versionSpecDtoFromPlan builds the version_spec client DTO from the plan, applying the manual-output
// override: manual building blocks have backend-derived outputs (left unknown in the plan), so the
// configured PLATFORM_TENANT_ID hints are sourced straight from config (see manualConfiguredOutputs).
// Non-manual implementations configure outputs explicitly and are left untouched. Shared by Create and
// Update; check diags for errors after calling.
func versionSpecDtoFromPlan(ctx context.Context, plan buildingBlockDefinition, bbdUuid string, config generic.AttributeGetter, diags *diag.Diagnostics) client.MeshBuildingBlockDefinitionVersionSpec {
	dto := plan.VersionSpec.ToClientDto(bbdUuid)
	if dto.Implementation.Manual != nil {
		dto.Outputs = manualConfiguredOutputs(ctx, config, dto.Inputs, diags)
	}
	return dto
}

// manualOutputOverride reads a configured manual output with nullable fields, so an omitted attribute (nil)
// is distinguishable from an explicit value - notably display_order, where an omitted value must fall back to
// the derived positional order rather than pinning position 0.
type manualOutputOverride struct {
	DisplayName    *string                                                 `tfsdk:"display_name"`
	AssignmentType *client.MeshBuildingBlockDefinitionOutputAssignmentType `tfsdk:"assignment_type"`
	Type           *client.MeshBuildingBlockIOType                         `tfsdk:"type"`
	DisplayOrder   *int64                                                  `tfsdk:"display_order"`
}

// manualConfiguredOutputs synthesizes the FULL one-per-input output set to send to the backend, applying the
// user's sparse overrides on top of the backend-derived defaults (type translated from the input, display_name
// from the input, assignment_type NONE, positional display_order). Sending the full set is required because the
// backend is stateful: on update it preserves any previously stored override for an output key absent from the
// request, so a dropped override would otherwise linger. By sending every key explicitly - untracked ones reset
// to their derived values - the provider fully controls the result and removing an override by dropping it from
// config works. The output type is always derived because the backend rejects an empty or mismatching type.
func manualConfiguredOutputs(ctx context.Context, config generic.AttributeGetter, inputs map[string]*client.MeshBuildingBlockDefinitionInput, diags *diag.Diagnostics) map[string]client.MeshBuildingBlockDefinitionOutput {
	overrides := generic.GetAttribute[map[string]manualOutputOverride](
		ctx, config, path.Root("version_spec").AtName("outputs"), diags)
	if diags.HasError() {
		return nil
	}

	// Derived display_order mirrors the backend: the input's position in (display_order, key)-sorted order.
	keys := make([]string, 0, len(inputs))
	for key := range inputs {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(a, b string) int {
		if c := cmp.Compare(inputs[a].DisplayOrder, inputs[b].DisplayOrder); c != 0 {
			return c
		}
		return cmp.Compare(a, b)
	})

	none := client.MeshBuildingBlockDefinitionOutputAssignmentTypeNone.Unwrap()
	outputs := make(map[string]client.MeshBuildingBlockDefinitionOutput, len(inputs))
	for index, key := range keys {
		input := inputs[key]
		output := client.MeshBuildingBlockDefinitionOutput{
			DisplayName:    input.DisplayName,
			Type:           translateManualInputTypeToOutput(input.Type),
			AssignmentType: none,
			DisplayOrder:   int64(index),
		}
		if override, ok := overrides[key]; ok {
			if override.DisplayName != nil && *override.DisplayName != "" {
				output.DisplayName = *override.DisplayName
			}
			if override.AssignmentType != nil && *override.AssignmentType != "" {
				output.AssignmentType = *override.AssignmentType
			}
			if override.DisplayOrder != nil {
				output.DisplayOrder = *override.DisplayOrder
			}
		}
		outputs[key] = output
	}
	return outputs
}

func (r *buildingBlockDefinitionResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		// do nothing in case of delete
		return
	}

	versionSpecSecretsChanged := false
	secret.WalkSecretPathsIn(req.Plan.Raw, &resp.Diagnostics, func(attributePath path.Path, diags *diag.Diagnostics) {
		versionChanged := secret.SetToUnknownIfVersionChangedOrCreated(ctx, req.Plan, req.State, &resp.Plan)(attributePath, diags)
		if versionChanged {
			if steps := attributePath.Steps(); len(steps) > 0 {
				if steps[0].Equal(path.PathStepAttributeName("version_spec")) {
					versionSpecSecretsChanged = true
				}
			}
		}
	})
	if resp.Diagnostics.HasError() {
		return
	}

	if !req.State.Raw.IsKnown() || req.State.Raw.IsNull() {
		// Create needs no output handling. Manual outputs are Computed and derived by the backend at apply;
		// the create plan leaves them known-after-apply and the read-back prunes to the tracked overrides, so
		// the create plan is consistent with the apply result.
		return
	}

	// Modifying only a certain portion of the plan related to output versions
	type buildingBlockDefinitionPartial struct {
		VersionSpec struct {
			Draft         generic.NullIsUnknown[bool]                                           `tfsdk:"draft"`
			VersionNumber int64                                                                 `tfsdk:"version_number"`
			State         generic.NullIsUnknown[client.MeshBuildingBlockDefinitionVersionState] `tfsdk:"state"`
		} `tfsdk:"version_spec"`
		Versions             []buildingBlockDefinitionVersionRef                       `tfsdk:"versions"`
		VersionLatest        buildingBlockDefinitionVersionRef                         `tfsdk:"version_latest"`
		VersionLatestRelease generic.NullIsUnknown[*buildingBlockDefinitionVersionRef] `tfsdk:"version_latest_release"`
	}

	state := generic.Get[buildingBlockDefinitionPartial](ctx, req.State, &resp.Diagnostics)
	plan := generic.Get[buildingBlockDefinitionPartial](ctx, req.Plan, &resp.Diagnostics, generic.WithSetUnknownValueToZero())
	if resp.Diagnostics.HasError() {
		return
	}

	// Creating or rotating a secret changes the version_spec (the secret machinery above already detected
	// this as versionSpecSecretsChanged). On a released (non-draft) version that stays released, the
	// version_spec is immutable, so reject it here at plan time with a clear, actionable error instead of
	// letting the content-hash safeguard fail opaquely on the planned plaintext later in Update (issue #196).
	// Flipping draft=false->true (plan draft becomes true) creates a new draft version and is allowed.
	planDraftKnownReleased := plan.VersionSpec.Draft.Value != nil && !plan.VersionSpec.Draft.Get()
	if versionSpecSecretsChanged && !state.VersionSpec.Draft.Get() && planDraftKnownReleased {
		resp.Diagnostics.AddError("Error updating version_spec", fmt.Sprintf(
			"Updating a version_spec in non-draft state is not allowed. "+
				"Rotating a secret on the released version %d changes the version_spec, which is immutable. "+
				"Set version_spec.draft = true to create a new draft version; the secret rotation can be applied in the same step.",
			state.VersionLatest.Number.Get(),
		))
		return
	}

	// Manual building blocks track only the user's output overrides (a sparse subset); the backend derives
	// the rest. A declared override's display_name/type/display_order are Computed and held from state by
	// UseStateForUnknown, so a no-op plan stays fully known and the content hash is stable. But the backend
	// re-derives type and positional display_order whenever the inputs change OR the tracked-override key set
	// changes (including dropping all overrides), so in those cases leave the whole outputs map - and the
	// content hash that includes it - unknown, and let apply reconcile.
	// Non-manual implementations configure outputs explicitly and the backend does not derive them.
	outputsPath := path.Root("version_spec").AtName("outputs")
	inputsPath := path.Root("version_spec").AtName("inputs")
	manualPath := path.Root("version_spec").AtName("implementation").AtName("manual")
	versionSpecOutputsUncertain := false
	var manual types.Object
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, manualPath, &manual)...)
	if !manual.IsNull() && !manual.IsUnknown() {
		var configOutputs, stateOutputs, planInputs, stateInputs types.Map
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, outputsPath, &configOutputs)...)
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, outputsPath, &stateOutputs)...)
		resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, inputsPath, &planInputs)...)
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, inputsPath, &stateInputs)...)
		if resp.Diagnostics.HasError() {
			return
		}
		inputsChanged := !planInputs.Equal(stateInputs)
		switch {
		case configOutputs.IsNull():
			// Outputs omitted: the whole set is derived. Leave it unknown when the derived result cannot be
			// predicted - the inputs changed, or overrides are being dropped (state still holds some) - so apply
			// reconciles. When nothing changed, leaving the state value avoids a perpetual "known after apply".
			stateEmpty := stateOutputs.IsNull() || len(stateOutputs.Elements()) == 0
			if inputsChanged || !stateEmpty {
				versionSpecOutputsUncertain = true
				resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, outputsPath, types.MapUnknown(stateOutputs.ElementType(ctx)))...)
			}
		case inputsChanged:
			// Overrides are declared and the inputs changed, so the backend re-derives each output's type and
			// positional display_order. The whole configured map cannot be planned unknown, so null only the
			// Computed fields held from state that a re-derivation can shift - type (never configured for
			// manual), and display_name/display_order where the user did not set them - and defer the hash.
			versionSpecOutputsUncertain = true
			for key, elem := range configOutputs.Elements() {
				obj, ok := elem.(types.Object)
				if !ok {
					continue
				}
				attrs := obj.Attributes()
				base := outputsPath.AtMapKey(key)
				resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, base.AtName("type"), types.StringUnknown())...)
				if dn := attrs["display_name"]; dn == nil || dn.IsNull() {
					resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, base.AtName("display_name"), types.StringUnknown())...)
				}
				if do := attrs["display_order"]; do == nil || do.IsNull() {
					resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, base.AtName("display_order"), types.Int64Unknown())...)
				}
			}
		}
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Determine this very carefully and leave it unknown if the underlying version_spec has unknown values
	// somewhere deep down. Read from resp.Plan, not req.Plan: for a manual building block the outputs were
	// just reconciled into resp.Plan above (reused from state, or left unknown), whereas req.Plan still holds
	// the raw configured outputs - a subset with the schema-default display_order 0. Hashing req.Plan would
	// predict a content_hash that disagrees with the one the backend-derived outputs produce at apply,
	// surfacing as "inconsistent result after apply" on the content_hash of a released or re-drafted version.
	versionSpecContentHash := func() (result generic.NullIsUnknown[string]) {
		if versionSpecSecretsChanged || versionSpecOutputsUncertain {
			return
		}
		versionSpecPath := path.Root("version_spec")
		var versionSpec types.Object
		resp.Plan.GetAttribute(ctx, versionSpecPath, &versionSpec)
		attributes := versionSpec.Attributes()
		attributeTypes := versionSpec.AttributeTypes(ctx)
		delete(attributes, "draft")
		delete(attributeTypes, "draft")
		tfValue, err := types.ObjectValueMust(attributeTypes, attributes).ToTerraformValue(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Converting version_spec to Terraform value failed", err.Error())
			return
		}
		if tfValue.IsFullyKnown() {
			result.Value = new(calculateBuildingBlockDefinitionVersionContentHash(
				generic.GetAttribute[client.MeshBuildingBlockDefinitionVersionSpec](
					ctx, resp.Plan, versionSpecPath, &resp.Diagnostics,
					buildingBlockDefinitionVersionConverterOptions(ctx, req.Config, req.Plan, req.State)...),
				&resp.Diagnostics,
			).toBase64())
		}
		return
	}()
	if resp.Diagnostics.HasError() {
		return
	}

	// always start from the current state
	plan.VersionLatestRelease = state.VersionLatestRelease
	plan.Versions = state.Versions
	plan.VersionLatest = state.VersionLatest

	defer func() {
		plan.VersionLatest = plan.Versions[len(plan.Versions)-1]
		resp.Diagnostics.Append(generic.SetPartial(ctx, &resp.Plan, plan)...)
	}()

	if plan.VersionSpec.Draft.Value == nil {
		// edge case that the draft state is computed/unknown during plan
		// then just return for now and keep state, but mark version_latest as unknown just to be sure
		resp.Diagnostics.AddWarning("version_spec.draft flag unknown",
			"If version_spec.draft is unknown, it cannot be determined if a new version is released. "+
				"Please make that flag available at plan time.")
		plan.Versions[len(plan.Versions)-1] = buildingBlockDefinitionVersionRef{} // all unknown
		plan.Versions[len(plan.Versions)-1].ContentHash = versionSpecContentHash  // possibly also unknown
		return
	}

	switch {
	case !state.VersionSpec.Draft.Get() && plan.VersionSpec.Draft.Get():
		// changing draft=false->true means creating a new draft version from the existing one with increased version number
		plan.VersionSpec.State = generic.KnownValue(client.MeshBuildingBlockDefinitionVersionStateDraft.Unwrap())
		plan.VersionSpec.VersionNumber = state.VersionSpec.VersionNumber + 1
		plan.Versions = append(state.Versions, buildingBlockDefinitionVersionRef{
			Uuid:        generic.NullIsUnknown[string]{},
			Number:      generic.KnownValue(plan.VersionSpec.VersionNumber),
			State:       generic.KnownValue(client.MeshBuildingBlockDefinitionVersionStateDraft.Unwrap()),
			ContentHash: versionSpecContentHash,
		})
	case state.VersionSpec.Draft.Get():
		// State is in draft=true, and plan might have draft=false
		if !plan.VersionSpec.Draft.Get() {
			// Draft changes to false according to plan (from draft=true in state)
			// If the BBD is part of a non-admin (non-partner) workspace, the BBD does change to RELEASED state immediately, but needs review approval first.
			// Thus, we need to be defensive here and keep values unknown for know.
			// SetFromVersionClientDtos detects this an issues a warning if the release in pending.
			plan.VersionSpec.State = generic.NullIsUnknown[client.MeshBuildingBlockDefinitionVersionState]{}
			plan.VersionLatestRelease = generic.NullIsUnknown[*buildingBlockDefinitionVersionRef]{}
		}

		plan.Versions = slices.Clone(state.Versions)
		latestVersionRef := &plan.Versions[len(plan.Versions)-1]

		latestVersionRef.State = plan.VersionSpec.State
		latestVersionRef.ContentHash = versionSpecContentHash
	}
}

func (r *buildingBlockDefinitionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	converterOptions := buildingBlockDefinitionConverterOptions().Append(buildingBlockDefinitionVersionConverterOptions(ctx, req.Config, req.Plan, req.State)...)
	state := generic.Get[buildingBlockDefinition](ctx, req.State, &resp.Diagnostics, converterOptions...)
	plan := generic.Get[buildingBlockDefinition](ctx, req.Plan, &resp.Diagnostics, converterOptions.Append(generic.WithSetUnknownValueToZero())...)
	if resp.Diagnostics.HasError() {
		return
	}
	bbdUuid := *state.Metadata.Uuid
	if resp.Diagnostics.HasError() {
		return
	}

	updatedDto, err := r.buildingBlockDefinitionClient.Update(ctx, bbdUuid, client.MeshBuildingBlockDefinition{
		Metadata: plan.Metadata,
		Spec:     plan.Spec,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating MeshBuildingBlockDefinition", err.Error())
		return
	}

	// Keep the tags the user declared rather than the superset the API returns, mirroring Create.
	plannedTags := plan.Metadata.Tags
	plan.SetFromClientDto(updatedDto, &resp.Diagnostics)
	plan.Metadata.Tags = plannedTags
	resp.Diagnostics.Append(generic.SetAttributeTo(ctx, &resp.State, path.Root("metadata"), plan.Metadata, buildingBlockDefinitionConverterOptions()...)...)
	resp.Diagnostics.Append(generic.SetAttributeTo(ctx, &resp.State, path.Root("spec"), plan.Spec, buildingBlockDefinitionConverterOptions()...)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Handle version_spec update logic

	versionSpecDto := versionSpecDtoFromPlan(ctx, plan, bbdUuid, req.Config, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	var updatedVersionDto *client.MeshBuildingBlockDefinitionVersion
	switch {
	case !state.VersionSpec.Draft && plan.VersionSpec.Draft:
		// changing draft=false->true means creating a new draft version from the existing one with increased version number
		versionSpecDto.VersionNumber = new(state.VersionLatest.Number.Get() + 1)
		updatedVersionDto, err = r.buildingBlockDefinitionVersionClient.Create(ctx, plan.Metadata.OwnedByWorkspace, versionSpecDto)
		if err != nil {
			resp.Diagnostics.AddError("Error creating new version", fmt.Sprintf(
				"Failed to create new version for building block '%s', ID=%s:\n%s",
				updatedDto.Spec.DisplayName, bbdUuid, err.Error(),
			))
			return
		}
	case !state.VersionSpec.Draft:
		// state (and plan) are in draft=false (aka released), so one should not change version_spec at all
		// this makes released or in-review versions immutable. A secret rotation on a released version is
		// already rejected at plan time in ModifyPlan with a clear message (issue #196), so it never reaches
		// here; any remaining version_spec change is caught by the content-hash comparison below.
		versionSpecDtoContentHash := calculateBuildingBlockDefinitionVersionContentHash(versionSpecDto, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}

		cmp := versionSpecDtoContentHash.compareToStored(state.VersionLatest.ContentHash.Get())
		if cmp == hashIncomparable {
			// The stored hash was produced by a different/legacy algorithm version (state written by an
			// older provider and never refreshed, imported, or planned with -refresh=false), so its value
			// cannot be compared against the current-version hash. Recompute the released version's hash
			// from the authoritative spec in state AT THE CURRENT version so we compare.
			// We build the DTO straight from state.VersionSpec.
			stateSpecDto := state.VersionSpec.ToClientDto(bbdUuid)
			authoritative := calculateBuildingBlockDefinitionVersionContentHash(stateSpecDto, &resp.Diagnostics)
			if resp.Diagnostics.HasError() {
				return
			}
			cmp = versionSpecDtoContentHash.compareToStored(authoritative.toBase64())
		}

		switch cmp {
		case hashSame:
			// state is in draft=false (aka released), and there's no indication of change in version_specs,
			// so all is good, and we don't need to do anything with the backend
			return
		default: // hashDifferent, or still-incomparable -> fail safe by rejecting a change to an immutable version
			resp.Diagnostics.AddError("Error updating version_spec", fmt.Sprintf(
				"Updating a version_spec in non-draft (released) state is not allowed — released versions are immutable.\n\n"+
					"To publish your changes as a new version, first set draft = true and apply to create a draft. "+
					"Once you are happy with the draft, set draft = false and apply again to release it.\n\n"+
					"(The content hash would change from %s to %s.)",
				state.VersionLatest.ContentHash.Get(), versionSpecDtoContentHash.toBase64(),
			))
			return
		}
	default:
		// State is in draft=true, so we update the version_spec (even if content hashes are equal)
		latestVersionUuid := *state.VersionLatest.Uuid.Value
		updatedVersionDto, err = r.buildingBlockDefinitionVersionClient.Update(ctx, latestVersionUuid, plan.Metadata.OwnedByWorkspace, versionSpecDto)
		if err != nil {
			resp.Diagnostics.AddError("Error updating version", fmt.Sprintf(
				"Failed to update version '%s' for building block '%s', ID=%s:\n%s",
				latestVersionUuid, updatedDto.Spec.DisplayName, bbdUuid, err.Error(),
			))
			return
		}
	}

	// Re-read all versions to update version refs
	allVersionDtos, err := r.buildingBlockDefinitionVersionClient.List(ctx, bbdUuid)
	if err != nil {
		resp.Diagnostics.AddError("Error listing versions after update", err.Error())
		return
	}

	plan.SetFromVersionClientDtos(&resp.Diagnostics, generic.KnownValue(plan.VersionSpec.Draft), bbdUuid, allVersionDtos...)
	if resp.Diagnostics.HasError() {
		return
	}
	if updatedVersionDto != nil {
		updatedVersionSpecContentHash := calculateBuildingBlockDefinitionVersionContentHash(updatedVersionDto.Spec, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		if updatedVersionSpecContentHash.compareToStored(plan.VersionLatest.ContentHash.Get()) == hashDifferent {
			resp.Diagnostics.AddError("Inconsistent content hash of version_spec after update", fmt.Sprintf(
				"The content hash of the latest version after listing does not match the content hash of the updated/created response: %s != %s. "+
					"This is most likely a bug in the backend.",
				plan.VersionLatest.ContentHash.Get(), updatedVersionSpecContentHash.toBase64(),
			))
			return
		}
	}

	// Finally, the plan is aligned with the backend, and we can set it as the new state!
	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, plan, converterOptions...)...)
}

func (r *buildingBlockDefinitionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	stateMetadata := generic.GetAttribute[client.MeshBuildingBlockDefinitionMetadata](ctx, req.State, path.Root("metadata"), &resp.Diagnostics, generic.WithSetUnknownValueToZero())
	if resp.Diagnostics.HasError() {
		return
	}
	if stateMetadata.Uuid == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	if err := r.buildingBlockDefinitionClient.Delete(ctx, *stateMetadata.Uuid); err != nil {
		resp.Diagnostics.AddError("Error deleting building block definition", err.Error())
	}
}

func (r *buildingBlockDefinitionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("metadata").AtName("uuid"), req, resp)
}
