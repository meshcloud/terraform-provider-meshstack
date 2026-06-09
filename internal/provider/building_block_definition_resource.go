package provider

import (
	"context"
	"fmt"
	"maps"
	"slices"

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

	// Set spec/metadata of BBD immediately after successful creation
	plan.SetFromClientDto(createdDto, &resp.Diagnostics)
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
	isDraft := generic.GetAttribute[generic.NullIsUnknown[bool]](ctx, req.State, path.Root("version_spec").AtName("draft"), &resp.Diagnostics, generic.WithSetUnknownValueToZero())
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
	state.SetFromVersionClientDtos(&resp.Diagnostics, isDraft, bbdUuid, versionDtos...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, state,
		buildingBlockDefinitionConverterOptions().Append(buildingBlockDefinitionVersionConverterOptions(ctx, nil, nil, req.State)...)...)...)
}

func (r *buildingBlockDefinitionResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	manualPath := path.Root("version_spec").AtName("implementation").AtName("manual")
	outputsPath := path.Root("version_spec").AtName("outputs")

	var manual types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, manualPath, &manual)...)
	var outputs types.Map
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, outputsPath, &outputs)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Manual building blocks derive their outputs from the inputs on the backend (one output per input,
	// assignment type NONE). The backend only honors outputs the user marks with assignment type
	// PLATFORM_TENANT_ID; everything else it regenerates. So configuring any non-PLATFORM_TENANT_ID output
	// for a manual building block is rejected here - omit it and let it be computed (see issues #131, #176).
	if manual.IsNull() || manual.IsUnknown() || outputs.IsNull() || outputs.IsUnknown() {
		return
	}
	for key, assignmentType := range outputAssignmentTypes(outputs) {
		if assignmentType == client.MeshBuildingBlockDefinitionOutputAssignmentTypePlatformTenantID.String() {
			continue
		}
		// An omitted assignment_type defaults to NONE, which the backend ignores just like an explicit
		// NONE, so reject it too instead of silently accepting config that does nothing.
		configured := fmt.Sprintf("assignment_type %q", assignmentType)
		if assignmentType == "" {
			configured = fmt.Sprintf("no assignment_type (defaults to %s)", client.MeshBuildingBlockDefinitionOutputAssignmentTypeNone)
		}
		resp.Diagnostics.AddAttributeError(
			outputsPath.AtMapKey(key),
			"manual building block outputs may only assign PLATFORM_TENANT_ID",
			fmt.Sprintf("Manual building block definitions derive their outputs from the inputs automatically. "+
				"Output %q has %s; remove it so it can be computed from the API response. "+
				"Only outputs with assignment_type %s may be configured (to mark which output carries the tenant id).",
				key, configured, client.MeshBuildingBlockDefinitionOutputAssignmentTypePlatformTenantID),
		)
	}
}

// outputAssignmentTypes returns the assignment_type of each output in the given map (keyed by output name),
// skipping null/unknown maps and elements. An omitted or null assignment_type yields an empty string (it
// defaults to NONE, so it must be surfaced rather than dropped); an unknown assignment_type is skipped
// because it cannot be validated yet.
func outputAssignmentTypes(outputs types.Map) map[string]string {
	result := map[string]string{}
	if outputs.IsNull() || outputs.IsUnknown() {
		return result
	}
	for key, elem := range outputs.Elements() {
		obj, ok := elem.(types.Object)
		if !ok || obj.IsNull() || obj.IsUnknown() {
			continue
		}
		assignmentType, ok := obj.Attributes()["assignment_type"].(types.String)
		if ok && assignmentType.IsUnknown() {
			continue
		}
		// A missing or null assignment_type defaults to NONE; represent it as "" so callers can reject it.
		if !ok || assignmentType.IsNull() {
			result[key] = ""
			continue
		}
		result[key] = assignmentType.ValueString()
	}
	return result
}

// platformTenantIdOutputKeysEqual reports whether the set of output keys assigned PLATFORM_TENANT_ID is
// the same in both maps. Used to detect whether a manual building block's configured tenant-id output
// changed, which (besides an inputs change) is the only way its reconciled outputs can change.
func platformTenantIdOutputKeysEqual(a, b types.Map) bool {
	tenantIdKeys := func(m types.Map) map[string]struct{} {
		keys := map[string]struct{}{}
		for key, assignmentType := range outputAssignmentTypes(m) {
			if assignmentType == client.MeshBuildingBlockDefinitionOutputAssignmentTypePlatformTenantID.String() {
				keys[key] = struct{}{}
			}
		}
		return keys
	}
	return maps.Equal(tenantIdKeys(a), tenantIdKeys(b))
}

// versionSpecDtoFromPlan builds the version_spec client DTO from the plan, applying the manual-output
// override: manual building blocks have backend-derived outputs (left unknown in the plan), so the
// configured PLATFORM_TENANT_ID hints are sourced straight from config (see manualConfiguredOutputs).
// Non-manual implementations configure outputs explicitly and are left untouched. Shared by Create and
// Update; check diags for errors after calling.
func versionSpecDtoFromPlan(ctx context.Context, plan buildingBlockDefinition, bbdUuid string, config generic.AttributeGetter, diags *diag.Diagnostics) client.MeshBuildingBlockDefinitionVersionSpec {
	dto := plan.VersionSpec.ToClientDto(bbdUuid)
	if dto.Implementation.Manual != nil {
		dto.Outputs = manualConfiguredOutputs(ctx, config, diags)
	}
	return dto
}

// manualConfiguredOutputs reads the user-configured version_spec.outputs (the PLATFORM_TENANT_ID hints)
// from config so they can be sent to the backend even though the planned outputs value is left unknown
// for manual building blocks. Returns an empty map when outputs are omitted.
func manualConfiguredOutputs(ctx context.Context, config generic.AttributeGetter, diags *diag.Diagnostics) map[string]client.MeshBuildingBlockDefinitionOutput {
	outputs := generic.GetAttribute[map[string]client.MeshBuildingBlockDefinitionOutput](
		ctx, config, path.Root("version_spec").AtName("outputs"), diags, generic.WithSetUnknownValueToZero())
	if outputs == nil {
		// The backend rejects a null outputs property, so send an empty map when none are configured.
		outputs = map[string]client.MeshBuildingBlockDefinitionOutput{}
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
		// do nothing more in case of create
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

	// Manual building blocks have backend-derived outputs: the backend mirrors every input into an output
	// (assignment type NONE), preserving only outputs the user marked PLATFORM_TENANT_ID (see issues #131
	// and #176, and ValidateConfig). We cannot fully predict the reconciled outputs at plan time, so
	// whenever the inputs or the configured PLATFORM_TENANT_ID outputs change we leave outputs - and the
	// content hash that includes them - unknown and let the apply reconcile them from the API response.
	// Otherwise we reuse the reconciled value from state, which also avoids a perpetual diff between the
	// (partial) configured outputs and the (full) stored outputs. Non-manual implementations configure
	// outputs explicitly and the backend does not derive them, so they are left untouched here.
	outputsPath := path.Root("version_spec").AtName("outputs")
	inputsPath := path.Root("version_spec").AtName("inputs")
	manualPath := path.Root("version_spec").AtName("implementation").AtName("manual")
	versionSpecOutputsUncertain := false
	var manual types.Object
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, manualPath, &manual)...)
	if !manual.IsNull() && !manual.IsUnknown() {
		var planInputs, stateInputs, configOutputs, stateOutputs types.Map
		resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, inputsPath, &planInputs)...)
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, inputsPath, &stateInputs)...)
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, outputsPath, &configOutputs)...)
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, outputsPath, &stateOutputs)...)
		if resp.Diagnostics.HasError() {
			return
		}
		versionSpecOutputsUncertain = !planInputs.Equal(stateInputs) ||
			!platformTenantIdOutputKeysEqual(configOutputs, stateOutputs)
		if versionSpecOutputsUncertain {
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, outputsPath, types.MapUnknown(stateOutputs.ElementType(ctx)))...)
		} else {
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, outputsPath, stateOutputs)...)
		}
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Determine this very carefully and leave it unknown if the underlying version_spec has unknown values somewhere deep down
	versionSpecContentHash := func() (result generic.NullIsUnknown[string]) {
		if versionSpecSecretsChanged || versionSpecOutputsUncertain {
			return
		}
		versionSpecPath := path.Root("version_spec")
		var versionSpec types.Object
		req.Plan.GetAttribute(ctx, versionSpecPath, &versionSpec)
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
			result.Value = new(versionContentHash(
				generic.GetAttribute[client.MeshBuildingBlockDefinitionVersionSpec](
					ctx, req.Plan, versionSpecPath, &resp.Diagnostics,
					buildingBlockDefinitionVersionConverterOptions(ctx, req.Config, req.Plan, req.State)...),
				&resp.Diagnostics,
			))
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

	plan.SetFromClientDto(updatedDto, &resp.Diagnostics)
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
		// this makes released or in-review versions immutable
		versionSpecDtoContentHash := versionContentHash(versionSpecDto, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		if versionSpecDtoContentHash == state.VersionLatest.ContentHash.Get() {
			// state is in draft=false (aka released), and there's no change in version_spec,
			// so all is good, and we don't need to do anything with the backend
			return
		} else {
			resp.Diagnostics.AddError("Error updating version_spec", fmt.Sprintf(
				"Updating a version_spec in non-draft state is not allowed. The content hash would change from %s to %s.",
				state.VersionLatest.ContentHash.Get(), versionSpecDtoContentHash,
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
		updatedVersionSpecContentHash := versionContentHash(updatedVersionDto.Spec, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		if plan.VersionLatest.ContentHash.Get() != updatedVersionSpecContentHash {
			resp.Diagnostics.AddError("Inconsistent content hash of version_spec after update", fmt.Sprintf(
				"The content hash of the latest version after listing does not match the content hash of the updated/created response: %s != %s. "+
					"This is most likely a bug in the backend.",
				plan.VersionLatest.ContentHash.Get(), updatedVersionSpecContentHash,
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
