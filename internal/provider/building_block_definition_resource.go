package provider

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/client/types/ptr"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/secret"
)

var (
	_ resource.Resource                = &buildingBlockDefinitionResource{}
	_ resource.ResourceWithConfigure   = &buildingBlockDefinitionResource{}
	_ resource.ResourceWithImportState = &buildingBlockDefinitionResource{}
	_ resource.ResourceWithModifyPlan  = &buildingBlockDefinitionResource{}
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
	converterOptions := buildingBlockDefinitionConverterOptions(ctx, req.Plan, nil).
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
	resp.Diagnostics.Append(generic.SetAttributeTo(ctx, &resp.State, path.Root("metadata"), plan.Metadata, buildingBlockDefinitionConverterOptions(ctx, req.Plan, nil)...)...)
	resp.Diagnostics.Append(generic.SetAttributeTo(ctx, &resp.State, path.Root("spec"), plan.Spec, buildingBlockDefinitionConverterOptions(ctx, req.Plan, nil)...)...)
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
	createVersionSpecDto := plan.VersionSpec.ToClientDto(bbdUuid)
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
		buildingBlockDefinitionConverterOptions(ctx, nil, req.State).Append(buildingBlockDefinitionVersionConverterOptions(ctx, nil, nil, req.State)...)...)...)
}

func (r *buildingBlockDefinitionResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		// do nothing in case of delete
		return
	}

	versionSpecSecretsChanged := false
	secret.WalkSecretPathsIn(req.Plan.Raw, &resp.Diagnostics, func(attributePath path.Path, diags *diag.Diagnostics) {
		versionChanged := secret.SetHashToUnknownIfVersionChanged(ctx, req.Plan, req.State, &resp.Plan)(attributePath, diags)
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

	// Determine this very carefully and leave it unknown if the underlying version_spec has unknown values somewhere deep down
	versionSpecContentHash := func() (result generic.NullIsUnknown[string]) {
		if versionSpecSecretsChanged {
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
			versionSpecDtoContentHash := versionContentHash(
				generic.GetAttribute[client.MeshBuildingBlockDefinitionVersionSpec](
					ctx, req.Plan, versionSpecPath, &resp.Diagnostics,
					buildingBlockDefinitionVersionConverterOptions(ctx, req.Config, req.Plan, req.State)...),
				&resp.Diagnostics,
			)
			result.Value = &versionSpecDtoContentHash
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
	converterOptions := buildingBlockDefinitionConverterOptions(ctx, req.Plan, req.State).Append(buildingBlockDefinitionVersionConverterOptions(ctx, req.Config, req.Plan, req.State)...)
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
	resp.Diagnostics.Append(generic.SetAttributeTo(ctx, &resp.State, path.Root("metadata"), plan.Metadata, buildingBlockDefinitionConverterOptions(ctx, req.Plan, req.State)...)...)
	resp.Diagnostics.Append(generic.SetAttributeTo(ctx, &resp.State, path.Root("spec"), plan.Spec, buildingBlockDefinitionConverterOptions(ctx, req.Plan, req.State)...)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Handle version_spec update logic

	versionSpecDto := plan.VersionSpec.ToClientDto(bbdUuid)

	var updatedVersionDto *client.MeshBuildingBlockDefinitionVersion
	switch {
	case !state.VersionSpec.Draft && plan.VersionSpec.Draft:
		// changing draft=false->true means creating a new draft version from the existing one with increased version number
		versionSpecDto.VersionNumber = ptr.To(state.VersionLatest.Number.Get() + 1)
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
