package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/secret"
)

var (
	_ resource.Resource                = &buildingBlockDefinitionResource{}
	_ resource.ResourceWithConfigure   = &buildingBlockDefinitionResource{}
	_ resource.ResourceWithImportState = &buildingBlockDefinitionResource{}
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
	var plan buildingBlockDefinition
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createRequest := client.MeshBuildingBlockDefinition{
		Metadata: plan.Metadata.ToClientDto(&resp.Diagnostics),
		Spec:     plan.Spec,
	}
	if resp.Diagnostics.HasError() {
		return
	}
	createdDto, err := r.buildingBlockDefinitionClient.Create(ctx, createRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error creating MeshBuildingBlockDefinition", err.Error())
		return
	}
	bbdUuid := *createdDto.Metadata.Uuid
	plan.Metadata.SetFromClientDto(createdDto.Metadata, &resp.Diagnostics)
	plan.Spec = createdDto.Spec

	// Set spec/metadata of BBD immediately after successful creation
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata"), plan.Metadata)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec"), plan.Spec)...)
	if resp.Diagnostics.HasError() {
		return
	}

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

	versionSpecDto := plan.VersionSpec.ToClientDto(bbdUuid, secret.NewPlaintextSupplierForCreate(ctx, req), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Updating the empty created version with provided configuration to complete creation
	versionUuid := createdEmptyVersion.Metadata.Uuid
	createdVersionDto, err := r.buildingBlockDefinitionVersionClient.Update(ctx, versionUuid, versionSpecDto)
	if err != nil {
		resp.Diagnostics.AddError("Error updating initial version", fmt.Sprintf(
			"Building block '%s', ID=%s was just created, and the initial version '%s' failed to update with given version_spec configuration. "+
				"Most likely schema validation is insufficient and the API received an invalid or incomplete JSON payload.\n"+
				"Error: %s",
			createdDto.Spec.DisplayName, bbdUuid, versionUuid, err.Error(),
		))
		return
	}

	plan.VersionSpec.SetFromClientDto(createdVersionDto.Spec, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError("Error converting initial version to model", fmt.Sprintf(
			"Building block '%s', ID=%s was just created, and the updated initial version '%s' cannot be converted to the version_spec model. "+
				"Most likely schema validation is insufficient and the API received an invalid or incomplete JSON payload.",
			createdDto.Spec.DisplayName, bbdUuid, versionUuid,
		))
		return
	}
	plan.SetVersionRefsFromClientDto(ctx, &resp.Diagnostics, *createdVersionDto)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *buildingBlockDefinitionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state buildingBlockDefinition
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	bbdUuid := state.Metadata.Uuid.Get(&resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	readDefinition, err := r.buildingBlockDefinitionClient.Read(ctx, bbdUuid)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get building block definition", fmt.Sprintf("Reading the existing BBD '%s' failed: %s", bbdUuid, err.Error()))
		return
	} else if readDefinition == nil {
		resp.State.RemoveResource(ctx)
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
			readDefinition.Spec.DisplayName, bbdUuid,
		))
	}
	latestVersionDto := state.SetVersionRefsFromClientDto(ctx, &resp.Diagnostics, versionDtos...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.VersionSpec.SetFromClientDto(latestVersionDto.Spec, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError("Error converting latest version to model", fmt.Sprintf(
			"The latest version '%s' cannot be converted to the version_spec model. "+
				"Most likely schema validation is insufficient and the API received an invalid or incomplete JSON payload.",
			latestVersionDto.Metadata.Uuid,
		))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *buildingBlockDefinitionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan buildingBlockDefinition
	var state buildingBlockDefinition
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bbdUuid := state.Metadata.Uuid.Get(&resp.Diagnostics)
	bbdLatestVersion := generic.FromObject[buildingBlockDefinitionVersionRef](ctx, state.VersionLatest, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	updateRequest := client.MeshBuildingBlockDefinition{
		Metadata: plan.Metadata.ToClientDto(&resp.Diagnostics),
		Spec:     plan.Spec,
	}
	if resp.Diagnostics.HasError() {
		return
	}

	updatedDto, err := r.buildingBlockDefinitionClient.Update(ctx, bbdUuid, updateRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error updating MeshBuildingBlockDefinition", err.Error())
		return
	}
	plan.Metadata.SetFromClientDto(updatedDto.Metadata, &resp.Diagnostics)
	plan.Spec = updatedDto.Spec

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata"), plan.Metadata)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec"), plan.Spec)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Handle version_spec update logic

	versionSpecDto := plan.VersionSpec.ToClientDto(bbdUuid, secret.NewPlaintextSupplierForUpdate(ctx, req), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	var updatedVersionDto *client.MeshBuildingBlockDefinitionVersion
	switch {
	case !state.VersionSpec.Draft && plan.VersionSpec.Draft:
		// changing draft=false->true means creating a new draft version from the existing one with increased version number
		versionSpecDto.VersionNumber = clientTypes.PtrTo(bbdLatestVersion.Number + 1)
		updatedVersionDto, err = r.buildingBlockDefinitionVersionClient.Create(ctx, versionSpecDto)
		if err != nil {
			resp.Diagnostics.AddError("Error creating new version", fmt.Sprintf(
				"Failed to create new version for building block '%s', ID=%s:\n%s",
				updatedDto.Spec.DisplayName, bbdUuid, err.Error(),
			))
			return
		}
	case !state.VersionSpec.Draft:
		// state (and plan) are in draft=false (aka released), so one should not change version_spec at all
		// this makes released versions immutable
		versionSpecDtoContentHash, err := versionContentHash(versionSpecDto)
		if err != nil {
			resp.Diagnostics.AddError("Failed to determine content hash", fmt.Sprintf(
				"Content hashing of planned version_spec as client DTO failed: %s", err.Error(),
			))
			return
		}
		if versionSpecDtoContentHash == bbdLatestVersion.ContentHash {
			// state is in draft=false (aka released), and there's no change in version_spec,
			// so all is good, and we don't need to do anything with the backend
			return
		} else {
			resp.Diagnostics.AddError("Error updating version_spec", fmt.Sprintf(
				"Updating a version_spec in state %s is not allowed. The content hash would change from %s to %s.",
				state.VersionSpec.State, bbdLatestVersion.ContentHash, versionSpecDtoContentHash,
			))
			return
		}
	default:
		// State is in draft=true, so we update the version_spec (even if content hashes are equal)
		updatedVersionDto, err = r.buildingBlockDefinitionVersionClient.Update(ctx, bbdLatestVersion.Uuid, versionSpecDto)
		if err != nil {
			resp.Diagnostics.AddError("Error updating version", fmt.Sprintf(
				"Failed to update version '%s' for building block '%s', ID=%s:\n%s",
				bbdLatestVersion.Uuid, updatedDto.Spec.DisplayName, bbdUuid, err.Error(),
			))
			return
		}
	}

	plan.VersionSpec.SetFromClientDto(updatedVersionDto.Spec, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError("Error converting updated version to model", fmt.Sprintf(
			"The updated version '%s' cannot be converted to the version_spec model.",
			updatedVersionDto.Metadata.Uuid,
		))
		return
	}

	// Re-read all versions to update version refs
	allVersionDtos, err := r.buildingBlockDefinitionVersionClient.List(ctx, bbdUuid)
	if err != nil {
		resp.Diagnostics.AddError("Error listing versions after update", err.Error())
		return
	}
	plan.SetVersionRefsFromClientDto(ctx, &resp.Diagnostics, allVersionDtos...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Finally, the plan is aligned with the backend, and we can set it as the new state!
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *buildingBlockDefinitionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model buildingBlockDefinition
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if model.Metadata.Uuid.IsUnknown() {
		resp.State.RemoveResource(ctx)
		return
	}
	bbdUuid := model.Metadata.Uuid.Get(&resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.buildingBlockDefinitionClient.Delete(ctx, bbdUuid); err != nil {
		resp.Diagnostics.AddError("Error deleting building block definition", err.Error())
	}
}

func (r *buildingBlockDefinitionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("metadata").AtName("uuid"), req, resp)
}
