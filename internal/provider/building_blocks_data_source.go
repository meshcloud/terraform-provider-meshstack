package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
)

var (
	_ datasource.DataSource              = &buildingBlocksDataSource{}
	_ datasource.DataSourceWithConfigure = &buildingBlocksDataSource{}
)

func NewBuildingBlocksDataSource() datasource.DataSource {
	return &buildingBlocksDataSource{}
}

type buildingBlocksDataSource struct {
	client client.MeshBuildingBlockV2Client
}

// buildingBlocksDataSourceModel is the root model: optional filters plus the resulting list.
// All filter fields are nil/empty when unset and are then omitted from the backend query.
type buildingBlocksDataSourceModel struct {
	WorkspaceIdentifier          *string `tfsdk:"workspace_identifier"`
	ProjectIdentifier            *string `tfsdk:"project_identifier"`
	PlatformIdentifier           *string `tfsdk:"platform_identifier"`
	Name                         *string `tfsdk:"name"`
	DefinitionUuid               *string `tfsdk:"definition_uuid"`
	VersionUuid                  *string `tfsdk:"version_uuid"`
	VersionNumber                *string `tfsdk:"version_number"`
	TenantUuid                   *string `tfsdk:"tenant_uuid"`
	TargetKind                   *string `tfsdk:"target_kind"`
	Status                       *string `tfsdk:"status"`
	ManagedByWorkspaceIdentifier *string `tfsdk:"managed_by_workspace_identifier"`
	ManagedByDefinitionUuid      *string `tfsdk:"managed_by_definition_uuid"`

	BuildingBlocks []buildingBlockListItem `tfsdk:"building_blocks"`
}

type buildingBlockListItem struct {
	Metadata  client.MeshBuildingBlockV2Metadata `tfsdk:"metadata"`
	Spec      buildingBlockReadSpec              `tfsdk:"spec"`
	Status    *client.MeshBuildingBlockV2Status  `tfsdk:"status"`
	AllInputs map[string]buildingBlockAllInput   `tfsdk:"all_inputs"`
}

func (d *buildingBlocksDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_building_blocks"
}

func (d *buildingBlocksDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.client = client.BuildingBlockV2
	})...)
}

func (d *buildingBlocksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	optionalString := func(md string) schema.StringAttribute {
		return schema.StringAttribute{MarkdownDescription: md, Optional: true}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "List building blocks, with optional filters. " +
			"Each returned building block is read-only and mirrors the `meshstack_building_block` resource " +
			"(`metadata`/`spec`/`status`/`all_inputs`).",
		Attributes: map[string]schema.Attribute{
			// ---- filters ----
			"workspace_identifier": optionalString("Only return building blocks owned by or assigned to this workspace."),
			"project_identifier":   optionalString("Only return building blocks in this project."),
			"platform_identifier":  optionalString("Only return building blocks on this platform (`<platformInstance>.<location>`)."),
			"name":                 optionalString("Only return building blocks with this exact name."),
			"definition_uuid":      optionalString("Only return building blocks created from the building block definition with this UUID (the definition, not a specific version)."),
			"version_uuid":         optionalString("Only return building blocks created from the building block definition version with this UUID."),
			"version_number": optionalString("Only return building blocks created from this building block definition version number. " +
				"Accepts a plain number (`1`) or a `v`-prefixed string (`v1`); the `v` is stripped server-side."),
			"tenant_uuid": optionalString("Only return building blocks targeting the tenant with this UUID."),
			"target_kind": schema.StringAttribute{
				MarkdownDescription: "Only return building blocks with this target kind. One of `meshTenant`, `meshWorkspace`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(client.MeshObjectKind.Tenant, client.MeshObjectKind.Workspace),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Only return building blocks in this execution status. One of " + client.BuildingBlockStatuses.Markdown() + ".",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(client.BuildingBlockStatuses.Strings()...),
				},
			},
			"managed_by_workspace_identifier": optionalString("Platform-operator scope: return building blocks created from definitions owned by this workspace. " +
				"Requires the `MANAGED_BUILDINGBLOCK_LIST` authority."),
			"managed_by_definition_uuid": optionalString("Platform-operator scope: return building blocks created from the definition owned by the caller with this UUID. " +
				"Requires the `MANAGED_BUILDINGBLOCK_LIST` authority."),

			// ---- result ----
			"building_blocks": schema.ListNestedAttribute{
				MarkdownDescription: "The building blocks matching the given filters.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"metadata": schema.SingleNestedAttribute{
							MarkdownDescription: "Building block metadata.",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"uuid":               computedString("UUID which uniquely identifies the building block."),
								"owned_by_workspace": computedString("The workspace containing this building block."),
							},
						},
						"spec":       buildingBlockReadSpecAttribute(),
						"status":     buildingBlockReadStatusAttribute(),
						"all_inputs": buildingBlockReadAllInputsAttribute(),
					},
				},
			},
		},
	}
}

func (d *buildingBlocksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config buildingBlocksDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	blocks, err := d.client.List(ctx, client.MeshBuildingBlockV2ListFilter{
		WorkspaceIdentifier:          config.WorkspaceIdentifier,
		ProjectIdentifier:            config.ProjectIdentifier,
		PlatformIdentifier:           config.PlatformIdentifier,
		Name:                         config.Name,
		DefinitionUuid:               config.DefinitionUuid,
		VersionUuid:                  config.VersionUuid,
		VersionNumber:                config.VersionNumber,
		TenantUuid:                   config.TenantUuid,
		TargetKind:                   config.TargetKind,
		Status:                       config.Status,
		ManagedByWorkspaceIdentifier: config.ManagedByWorkspaceIdentifier,
		ManagedByDefinitionUuid:      config.ManagedByDefinitionUuid,
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to list building blocks", err.Error())
		return
	}

	config.BuildingBlocks = make([]buildingBlockListItem, 0, len(blocks))
	for i := range blocks {
		bb := blocks[i]
		config.BuildingBlocks = append(config.BuildingBlocks, buildingBlockListItem{
			Metadata:  bb.Metadata,
			Spec:      buildingBlockReadSpecFromDto(bb.Spec),
			Status:    bb.Status,
			AllInputs: buildingBlockAllInputsFromDto(bb.Spec.Inputs, &resp.Diagnostics),
		})
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// generic.Set handles the client-specific types in the item models: clientTypes.Any (outputs
	// value) and clientTypes.Set (parent_building_block_refs). all_inputs sensitive values are already
	// reduced to a hash-only secret.HashOnly by buildAllInput, so no secret converter is needed.
	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, config,
		withValueFromConverterForClientTypeAny(),
		generic.WithSliceTypeAsSet(clientTypes.IsSet),
	)...)
}
