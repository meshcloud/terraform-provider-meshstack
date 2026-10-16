package provider

import (
	"cmp"
	"context"
	"errors"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/meshcloud/meshstack-cli/client"

	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
)

var (
	_ datasource.DataSource              = &buildingBlockDefinitionsDataSource{}
	_ datasource.DataSourceWithConfigure = &buildingBlockDefinitionsDataSource{}
)

func NewBuildingBlockDefinitionsDataSource() datasource.DataSource {
	return &buildingBlockDefinitionsDataSource{}
}

type buildingBlockDefinitionsDataSource struct {
	meshBuildingBlockDefinitionClient        client.MeshBuildingBlockDefinitionClient
	meshBuildingBlockDefinitionVersionClient client.MeshBuildingBlockDefinitionVersionClient
}

type buildingBlockDefinitionsDataSourceModel struct {
	WorkspaceIdentifier      *string                                  `tfsdk:"workspace_identifier"`
	Uuids                    []string                                 `tfsdk:"uuids"`
	BuildingBlockDefinitions []buildingBlockDefinitionDataSourceModel `tfsdk:"building_block_definitions"`
}

type buildingBlockDefinitionDataSourceModel struct {
	Metadata             buildingBlockDefinitionDataSourceMetadataModel              `tfsdk:"metadata"`
	Spec                 buildingBlockDefinitionDataSourceSpecModel                  `tfsdk:"spec"`
	Versions             map[string]buildingBlockDefinitionDataSourceVersionRefModel `tfsdk:"versions"`
	VersionLatest        buildingBlockDefinitionDataSourceVersionRefModel            `tfsdk:"version_latest"`
	VersionLatestRelease *buildingBlockDefinitionDataSourceVersionRefModel           `tfsdk:"version_latest_release"`
	Ref                  buildingBlockDefinitionRef                                  `tfsdk:"ref"`
}

type buildingBlockDefinitionDataSourceMetadataModel struct {
	Uuid             string `tfsdk:"uuid"`
	OwnedByWorkspace string `tfsdk:"owned_by_workspace"`
}

type buildingBlockDefinitionDataSourceSpecModel struct {
	DisplayName string                       `tfsdk:"display_name"`
	TargetType  client.MeshBuildingBlockType `tfsdk:"target_type"`
}

type buildingBlockDefinitionDataSourceVersionRefModel struct {
	Uuid        string  `tfsdk:"uuid"`
	Number      int64   `tfsdk:"number"`
	State       string  `tfsdk:"state"`
	ContentHash *string `tfsdk:"content_hash"`

	WorkloadIdentityFederation *client.MeshBuildingBlockDefinitionWif `tfsdk:"workload_identity_federation"`
}

func (d *buildingBlockDefinitionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_building_block_definitions"
}

func (d *buildingBlockDefinitionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.meshBuildingBlockDefinitionClient = client.BuildingBlockDefinition
		d.meshBuildingBlockDefinitionVersionClient = client.BuildingBlockDefinitionVersion
	})...)
}

func buildingBlockDefinitionDataSourceAttributes() map[string]schema.Attribute {
	versionRefAttributes := map[string]schema.Attribute{
		"uuid": schema.StringAttribute{
			MarkdownDescription: "UUID of the version.",
			Computed:            true,
		},
		"number": schema.Int64Attribute{
			MarkdownDescription: "Version number.",
			Computed:            true,
		},
		"state": schema.StringAttribute{
			MarkdownDescription: "State of the version.",
			Computed:            true,
		},
		"content_hash": schema.StringAttribute{
			MarkdownDescription: "Content hash of the version. Null when accessing cross-workspace definitions without sufficient permissions.",
			Computed:            true,
			Optional:            true,
		},
		"workload_identity_federation": dataSourceVersionWifAttribute(
			"The identity a run of this version presents to a cloud: the subject template of the version's runner, " +
				"filled in by meshStack for this definition. Grant cloud access to `subject`. " +
				"Null when the runner declares no workload identity federation, or when meshStack is older than 2026.40.0.",
		),
	}

	return map[string]schema.Attribute{
		"metadata": schema.SingleNestedAttribute{
			MarkdownDescription: "Building block definition metadata.",
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"uuid": schema.StringAttribute{
					MarkdownDescription: "UUID of the building block definition.",
					Computed:            true,
				},
				"owned_by_workspace": schema.StringAttribute{
					MarkdownDescription: "Workspace identifier owning this definition.",
					Computed:            true,
				},
			},
		},
		"spec": schema.SingleNestedAttribute{
			MarkdownDescription: "Key specification fields for selecting a definition.",
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"display_name": schema.StringAttribute{
					MarkdownDescription: "Display name of the definition.",
					Computed:            true,
				},
				"target_type": schema.StringAttribute{
					MarkdownDescription: "Target type (`TENANT_LEVEL` or `WORKSPACE_LEVEL`).",
					Computed:            true,
				},
			},
		},
		"version_latest": schema.SingleNestedAttribute{
			MarkdownDescription: "Latest version (including drafts). Useful for BB specs that may target drafts.",
			Computed:            true,
			Attributes:          versionRefAttributes,
		},
		"versions": schema.MapNestedAttribute{
			MarkdownDescription: "All available versions. " + versionsKeyDescription,
			Computed:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: versionRefAttributes,
			},
		},
		"version_latest_release": schema.SingleNestedAttribute{
			MarkdownDescription: "Latest released version. Null when no release exists yet.",
			Computed:            true,
			Attributes:          versionRefAttributes,
		},
		"ref": meshRefByUuid(meshRefOptions{Kind: client.MeshObjectKind.BuildingBlockDefinition, Description: "Reference to this building block definition (for dependency refs).", Output: true}),
	}
}

func (d *buildingBlockDefinitionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List building block definitions with optional workspace filter. " +
			"Prefer this plural data source with `one(...)` for reusable wiring in examples. " +
			"For each returned definition, this data source performs an additional API call to load all versions; " +
			"use `workspace_identifier` to narrow scope where possible. " +
			"\n\n" +
			"**Cross-Workspace Access**: When accessing building block definitions from workspaces other than your own " +
			"using a workspace-scoped API key, the `content_hash` attribute will be null as it requires detailed version " +
			"information that is not accessible without workspace permissions. The `versions`, `version_latest`, and " +
			"`version_latest_release` attributes will still be populated with uuid, number, state, and workload_identity_federation." +
			"\n\n" + previewDisclaimer(),
		Attributes: map[string]schema.Attribute{
			"workspace_identifier": schema.StringAttribute{
				MarkdownDescription: "Optional workspace identifier filter (maps to `ownedByWorkspace` query param).",
				Optional:            true,
			},
			"uuids": schema.SetAttribute{
				MarkdownDescription: "Optional filter: return only the definitions with these uuids. " +
					"The versions of the other definitions are then not loaded. A uuid that matches no definition is ignored.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"building_block_definitions": schema.ListNestedAttribute{
				MarkdownDescription: "Matching building block definitions.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: buildingBlockDefinitionDataSourceAttributes(),
				},
			},
		},
	}
}

func (d *buildingBlockDefinitionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data buildingBlockDefinitionsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	definitions, err := d.meshBuildingBlockDefinitionClient.List(ctx, data.WorkspaceIdentifier)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list building block definitions", err.Error())
		return
	}

	result := make([]buildingBlockDefinitionDataSourceModel, 0, len(definitions))
	for _, definition := range definitions {
		if definition.Metadata.Uuid == nil {
			resp.Diagnostics.AddError(
				"Building block definition UUID missing",
				"API returned a building block definition without metadata.uuid, which cannot be represented in this data source.",
			)
			return
		}
		if data.Uuids != nil && !slices.Contains(data.Uuids, *definition.Metadata.Uuid) {
			continue
		}

		model := buildingBlockDefinitionDataSourceModelFrom(ctx, d.meshBuildingBlockDefinitionVersionClient, definition, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		result = append(result, model)
	}

	slices.SortFunc(result, func(a, b buildingBlockDefinitionDataSourceModel) int {
		return cmp.Compare(a.Metadata.Uuid, b.Metadata.Uuid)
	})

	data.BuildingBlockDefinitions = result
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func buildingBlockDefinitionDataSourceModelFrom(ctx context.Context, versionClient client.MeshBuildingBlockDefinitionVersionClient, definition client.MeshBuildingBlockDefinition, diags *diag.Diagnostics) buildingBlockDefinitionDataSourceModel {
	redacted := definition.Status != nil && definition.Status.RedactedForNonOwnerAccess

	var versions []client.MeshBuildingBlockDefinitionVersion
	var err error
	if !redacted {
		versions, err = versionClient.List(ctx, *definition.Metadata.Uuid)
	}
	httpErr, isHttpErr := errors.AsType[client.HttpError](err)
	if redacted || (isHttpErr && httpErr.IsForbidden()) {
		return buildVersionRefsFromStatus(diags, definition)
	}

	if err != nil {
		diags.AddError(
			"Unable to list building block definition versions",
			err.Error(),
		)
		return buildingBlockDefinitionDataSourceModel{}
	}

	identities := workloadIdentitiesByVersion(definition.Status, *definition.Metadata.Uuid, diags)
	if diags.HasError() {
		return buildingBlockDefinitionDataSourceModel{}
	}
	var versionModel buildingBlockDefinition
	versionModel.SetFromVersionClientDtos(diags, deriveDraftFromLatestVersion(versions), *definition.Metadata.Uuid, identities, versions...)
	if diags.HasError() {
		return buildingBlockDefinitionDataSourceModel{}
	}

	model := buildingBlockDefinitionDataSourceModel{
		Metadata: buildingBlockDefinitionDataSourceMetadataModel{
			Uuid:             *definition.Metadata.Uuid,
			OwnedByWorkspace: definition.Metadata.OwnedByWorkspace,
		},
		Spec: buildingBlockDefinitionDataSourceSpecModel{
			DisplayName: definition.Spec.DisplayName,
			TargetType:  definition.Spec.TargetType,
		},
		Versions:      convertBBDVersionRefs(diags, "versions", ascendingVersions(versionModel.Versions)),
		VersionLatest: convertBBDVersionRef(diags, "version_latest", versionModel.VersionLatest),
		Ref:           versionModel.Ref,
	}
	if versionModel.VersionLatestRelease != nil {
		model.VersionLatestRelease = new(convertBBDVersionRef(diags, "version_latest_release", *versionModel.VersionLatestRelease))
	}
	return model
}

// buildVersionRefsFromStatus builds version refs from definition status when the full version spec is not
// accessible, which is the case for a workspace that may consume a definition but does not own it.
// Content hash will be empty since it requires the full version spec which is not available in status.
func buildVersionRefsFromStatus(diags *diag.Diagnostics, definition client.MeshBuildingBlockDefinition) buildingBlockDefinitionDataSourceModel {
	if definition.Status == nil {
		diags.AddError(
			"Building block definition status missing",
			"API returned a building block definition without status, which is required to build version references.",
		)
		return buildingBlockDefinitionDataSourceModel{}
	}

	status := definition.Status

	// Convert status.Versions to version refs (no content_hash available)
	ascending := make([]buildingBlockDefinitionDataSourceVersionRefModel, len(status.Versions))
	for i, v := range status.Versions {
		ascending[i] = buildingBlockDefinitionDataSourceVersionRefModel{
			Uuid:                       v.VersionUuid,
			Number:                     v.VersionNumber,
			State:                      string(v.State),
			ContentHash:                nil, // Not available without full version spec
			WorkloadIdentityFederation: v.WorkloadIdentityFederation,
		}
	}

	// Sort versions ascending by number
	slices.SortFunc(ascending, func(a, b buildingBlockDefinitionDataSourceVersionRefModel) int {
		return cmp.Compare(a.Number, b.Number)
	})

	// Build latest version ref
	latestStatusVersion := findStatusVersion(status.Versions, status.LatestVersionUuid)
	latestVersion := buildingBlockDefinitionDataSourceVersionRefModel{
		Uuid:                       status.LatestVersionUuid,
		Number:                     status.LatestVersion,
		State:                      string(latestStatusVersion.State),
		ContentHash:                nil, // Not available without full version spec
		WorkloadIdentityFederation: latestStatusVersion.WorkloadIdentityFederation,
	}

	// Build latest released version ref (if exists)
	var latestReleasedVersion *buildingBlockDefinitionDataSourceVersionRefModel
	if status.LatestReleasedVersionUuid != nil && status.LatestReleasedVersion != nil {
		latestReleasedVersion = &buildingBlockDefinitionDataSourceVersionRefModel{
			Uuid:                       *status.LatestReleasedVersionUuid,
			Number:                     *status.LatestReleasedVersion,
			State:                      string(client.MeshBuildingBlockDefinitionVersionStateReleased.Unwrap()),
			ContentHash:                nil, // Not available without full version spec
			WorkloadIdentityFederation: findStatusVersion(status.Versions, *status.LatestReleasedVersionUuid).WorkloadIdentityFederation,
		}
	}

	return buildingBlockDefinitionDataSourceModel{
		Metadata: buildingBlockDefinitionDataSourceMetadataModel{
			Uuid:             *definition.Metadata.Uuid,
			OwnedByWorkspace: definition.Metadata.OwnedByWorkspace,
		},
		Spec: buildingBlockDefinitionDataSourceSpecModel{
			DisplayName: definition.Spec.DisplayName,
			TargetType:  definition.Spec.TargetType,
		},
		Versions:             versionsByKey(ascending, buildingBlockDefinitionDataSourceVersionRefModel.number),
		VersionLatest:        latestVersion,
		VersionLatestRelease: latestReleasedVersion,
		Ref:                  newBuildingBlockDefinitionRef(*definition.Metadata.Uuid),
	}
}

// findStatusVersion finds a version by UUID in the status versions list, or returns the zero value.
func findStatusVersion(versions []client.MeshBuildingBlockDefinitionStatusVersion, uuid string) client.MeshBuildingBlockDefinitionStatusVersion {
	for _, v := range versions {
		if v.VersionUuid == uuid {
			return v
		}
	}
	return client.MeshBuildingBlockDefinitionStatusVersion{}
}

func convertBBDVersionRef(diags *diag.Diagnostics, field string, ref buildingBlockDefinitionVersionRef) buildingBlockDefinitionDataSourceVersionRefModel {
	if ref.Uuid.IsUnknown() || ref.Number.IsUnknown() || ref.State.IsUnknown() || ref.ContentHash.IsUnknown() || ref.WorkloadIdentityFederation.IsUnknown() {
		diags.AddError(
			"Building block definition version reference unknown",
			"The field "+field+" contains unknown values in data source read, which indicates inconsistent API/model conversion.",
		)
		return buildingBlockDefinitionDataSourceVersionRefModel{}
	}
	return buildingBlockDefinitionDataSourceVersionRefModel{
		Uuid:        ref.Uuid.Get(),
		Number:      ref.Number.Get(),
		State:       string(ref.State.Get()),
		ContentHash: new(ref.ContentHash.Get()),

		WorkloadIdentityFederation: ref.WorkloadIdentityFederation.Get(),
	}
}

func convertBBDVersionRefs(diags *diag.Diagnostics, field string, ascending []buildingBlockDefinitionVersionRef) map[string]buildingBlockDefinitionDataSourceVersionRefModel {
	converted := make([]buildingBlockDefinitionDataSourceVersionRefModel, len(ascending))
	for i, ref := range ascending {
		converted[i] = convertBBDVersionRef(diags, field, ref)
	}
	return versionsByKey(converted, buildingBlockDefinitionDataSourceVersionRefModel.number)
}

func (ref buildingBlockDefinitionDataSourceVersionRefModel) number() int64 {
	return ref.Number
}

func deriveDraftFromLatestVersion(versionDtos []client.MeshBuildingBlockDefinitionVersion) generic.NullIsUnknown[bool] {
	if len(versionDtos) == 0 {
		return generic.NullIsUnknown[bool]{}
	}
	latest := slices.MaxFunc(versionDtos, func(a, b client.MeshBuildingBlockDefinitionVersion) int {
		switch {
		case a.Spec.VersionNumber == nil:
			if b.Spec.VersionNumber == nil {
				return 0
			}
			return -1
		case b.Spec.VersionNumber == nil:
			return 1
		default:
			return cmp.Compare(*a.Spec.VersionNumber, *b.Spec.VersionNumber)
		}
	})
	if latest.Spec.State == nil {
		return generic.NullIsUnknown[bool]{}
	}
	return generic.KnownValue(*latest.Spec.State == client.MeshBuildingBlockDefinitionVersionStateDraft.Unwrap())
}
