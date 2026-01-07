package provider

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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
	BuildingBlockDefinition        client.MeshBuildingBlockDefinitionClient
	BuildingBlockDefinitionVersion client.MeshBuildingBlockDefinitionVersionClient
}

func (r *buildingBlockDefinitionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_building_block_definition"
}

func (r *buildingBlockDefinitionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		r.BuildingBlockDefinition = client.BuildingBlockDefinition
		r.BuildingBlockDefinitionVersion = client.BuildingBlockDefinitionVersion
	})...)
}

// mockComputedVersions creates dummy version data for testing
// Returns version_latest, version_latest_release, and versions list
func mockComputedVersions() (version1 buildingBlockDefinitionVersion, version2 buildingBlockDefinitionVersion, versions []buildingBlockDefinitionVersion) {
	// Version 1 (RELEASED)
	version1 = buildingBlockDefinitionVersion{
		Uuid:   "dummy-version-uuid-1",
		Number: int64(1),
		State:  "RELEASED",
	}

	// Version 2 (DRAFT)
	version2 = buildingBlockDefinitionVersion{
		Uuid:   "dummy-version-uuid-2",
		Number: int64(2),
		State:  "DRAFT",
	}

	// versions list contains both versions
	// version_latest is always the highest version number (v2 DRAFT)
	// version_latest_release is the latest RELEASED version (v1)
	versions = []buildingBlockDefinitionVersion{version1, version2}
	return
}

func (r *buildingBlockDefinitionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan buildingBlockDefinitionResourceModel

	// Dummy implementation - populate computed fields
	plan.Metadata.Uuid = "dummy-uuid-12345"
	plan.Metadata.CreatedOn = "2026-01-07T14:00:00.000Z"
	// Set version info
	plan.VersionLatest, plan.VersionLatestRelease, plan.Versions = mockComputedVersions()

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *buildingBlockDefinitionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state buildingBlockDefinitionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// For dummy resource, just keep existing state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *buildingBlockDefinitionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan buildingBlockDefinitionResourceModel
	var state buildingBlockDefinitionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve computed metadata fields
	plan.Metadata.Uuid = state.Metadata.Uuid
	plan.Metadata.CreatedOn = state.Metadata.CreatedOn
	plan.Metadata.MarkedForDeletionOn = state.Metadata.MarkedForDeletionOn
	plan.Metadata.MarkedForDeletionBy = state.Metadata.MarkedForDeletionBy

	// Set version info
	plan.VersionLatest, plan.VersionLatestRelease, plan.Versions = mockComputedVersions()

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *buildingBlockDefinitionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// No-op for dummy resource
}

func (r *buildingBlockDefinitionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("metadata").AtName("uuid"), req, resp)
}
