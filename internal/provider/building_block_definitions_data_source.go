package provider

import (
	"cmp"
	"context"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/meshcloud/terraform-provider-meshstack/client"
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
	BuildingBlockDefinitions []buildingBlockDefinitionDataSourceModel `tfsdk:"building_block_definitions"`
}

type buildingBlockDefinitionDataSourceModel struct {
	Metadata             buildingBlockDefinitionDataSourceMetadataModel     `tfsdk:"metadata"`
	Spec                 buildingBlockDefinitionDataSourceSpecModel         `tfsdk:"spec"`
	Versions             []buildingBlockDefinitionDataSourceVersionRefModel `tfsdk:"versions"`
	VersionLatest        buildingBlockDefinitionDataSourceVersionRefModel   `tfsdk:"version_latest"`
	VersionLatestRelease *buildingBlockDefinitionDataSourceVersionRefModel  `tfsdk:"version_latest_release"`
	Ref                  buildingBlockDefinitionRef                         `tfsdk:"ref"`
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
	Uuid        string `tfsdk:"uuid"`
	Number      int64  `tfsdk:"number"`
	State       string `tfsdk:"state"`
	ContentHash string `tfsdk:"content_hash"`
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

func (d *buildingBlockDefinitionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
			MarkdownDescription: "Content hash of the version.",
			Computed:            true,
		},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "List building block definitions with optional workspace filter. " +
			"Prefer this plural data source with `one(...)` for reusable wiring in examples. " +
			"For each returned definition, this data source performs an additional API call to load all versions; " +
			"use `workspace_identifier` to narrow scope where possible. " + previewDisclaimer(),
		Attributes: map[string]schema.Attribute{
			"workspace_identifier": schema.StringAttribute{
				MarkdownDescription: "Optional workspace identifier filter (maps to `workspaceIdentifier` query param).",
				Optional:            true,
			},
			"building_block_definitions": schema.ListNestedAttribute{
				MarkdownDescription: "Matching building block definitions.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
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
						"versions": schema.ListNestedAttribute{
							MarkdownDescription: "All available versions, sorted ascending by version number.",
							Computed:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: versionRefAttributes,
							},
						},
						"version_latest_release": schema.SingleNestedAttribute{
							MarkdownDescription: "Latest released version. Null when no release exists yet.",
							Computed:            true,
							Optional:            true,
							Attributes:          versionRefAttributes,
						},
						"ref": schema.SingleNestedAttribute{
							MarkdownDescription: "Reference to this building block definition (for dependency refs).",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"kind": schema.StringAttribute{
									MarkdownDescription: "The kind of the object. Always `meshBuildingBlockDefinition`.",
									Computed:            true,
								},
								"uuid": schema.StringAttribute{
									MarkdownDescription: "UUID of the building block definition.",
									Computed:            true,
								},
							},
						},
					},
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

		versions, err := d.meshBuildingBlockDefinitionVersionClient.List(ctx, *definition.Metadata.Uuid)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to list building block definition versions",
				err.Error(),
			)
			return
		}

		var versionModel buildingBlockDefinition
		versionModel.SetFromVersionClientDtos(&resp.Diagnostics, deriveDraftFromLatestVersion(versions), *definition.Metadata.Uuid, versions...)
		if resp.Diagnostics.HasError() {
			return
		}

		result = append(result, buildingBlockDefinitionDataSourceModel{
			Metadata: buildingBlockDefinitionDataSourceMetadataModel{
				Uuid:             *definition.Metadata.Uuid,
				OwnedByWorkspace: definition.Metadata.OwnedByWorkspace,
			},
			Spec: buildingBlockDefinitionDataSourceSpecModel{
				DisplayName: definition.Spec.DisplayName,
				TargetType:  definition.Spec.TargetType,
			},
			Versions:      convertBBDVersionRefs(&resp.Diagnostics, "versions", versionModel.Versions),
			VersionLatest: convertBBDVersionRef(&resp.Diagnostics, "version_latest", versionModel.VersionLatest),
			Ref:           versionModel.Ref,
		})
		if versionModel.VersionLatestRelease != nil {
			result[len(result)-1].VersionLatestRelease = new(convertBBDVersionRef(&resp.Diagnostics, "version_latest_release", *versionModel.VersionLatestRelease))
		}
		if resp.Diagnostics.HasError() {
			return
		}
	}

	slices.SortFunc(result, func(a, b buildingBlockDefinitionDataSourceModel) int {
		return cmp.Compare(a.Metadata.Uuid, b.Metadata.Uuid)
	})

	data.BuildingBlockDefinitions = result
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func convertBBDVersionRef(diags *diag.Diagnostics, field string, ref buildingBlockDefinitionVersionRef) buildingBlockDefinitionDataSourceVersionRefModel {
	if ref.Uuid.IsUnknown() || ref.Number.IsUnknown() || ref.State.IsUnknown() || ref.ContentHash.IsUnknown() {
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
		ContentHash: ref.ContentHash.Get(),
	}
}

func convertBBDVersionRefs(diags *diag.Diagnostics, field string, refs []buildingBlockDefinitionVersionRef) []buildingBlockDefinitionDataSourceVersionRefModel {
	converted := make([]buildingBlockDefinitionDataSourceVersionRefModel, len(refs))
	for i, ref := range refs {
		converted[i] = convertBBDVersionRef(diags, field, ref)
	}
	return converted
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
