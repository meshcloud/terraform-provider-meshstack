package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/secret"
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

// buildingBlockListItem is the read-only, bbv3-aligned view of a single building block:
// metadata / spec / status / all_inputs. It intentionally omits the resource's writable
// spec.inputs; all backend inputs (with sensitive values reduced to a hash) are surfaced in
// all_inputs instead.
type buildingBlockListItem struct {
	Metadata  client.MeshBuildingBlockV2Metadata `tfsdk:"metadata"`
	Spec      buildingBlockListItemSpec          `tfsdk:"spec"`
	Status    *client.MeshBuildingBlockV2Status  `tfsdk:"status"`
	AllInputs map[string]buildingBlockAllInput   `tfsdk:"all_inputs"`
}

type buildingBlockListItemSpec struct {
	DisplayName                       string                                          `tfsdk:"display_name"`
	BuildingBlockDefinitionVersionRef buildingBlockListItemVersionRef                 `tfsdk:"building_block_definition_version_ref"`
	TargetRef                         client.MeshBuildingBlockV2TargetRef             `tfsdk:"target_ref"`
	ParentBuildingBlocks              clientTypes.Set[client.MeshBuildingBlockParent] `tfsdk:"parent_building_blocks"`
}

type buildingBlockListItemVersionRef struct {
	Uuid string `tfsdk:"uuid"`
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
	computedString := func(md string) schema.StringAttribute {
		return schema.StringAttribute{MarkdownDescription: md, Computed: true}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "List building blocks, with optional filters. " +
			"Each returned building block is read-only and mirrors the `meshstack_building_block` resource " +
			"(`metadata`/`spec`/`status`/`all_inputs`)." + previewDisclaimer(),
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
						"spec": schema.SingleNestedAttribute{
							MarkdownDescription: "Building block specification.",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"display_name": computedString("Display name for the building block as shown in meshPanel."),
								"building_block_definition_version_ref": schema.SingleNestedAttribute{
									MarkdownDescription: "References the building block definition version this building block is based on.",
									Computed:            true,
									Attributes: map[string]schema.Attribute{
										"uuid": computedString("UUID of the building block definition version."),
									},
								},
								"target_ref": schema.SingleNestedAttribute{
									MarkdownDescription: "References the building block target, a workspace or a tenant depending on the definition.",
									Computed:            true,
									Attributes: map[string]schema.Attribute{
										"kind": computedString("Target kind, one of `meshTenant`, `meshWorkspace`."),
										"uuid": computedString("UUID of the target tenant (for `meshTenant` targets)."),
										"name": computedString("Identifier of the target workspace (for `meshWorkspace` targets)."),
									},
								},
								"parent_building_blocks": schema.SetNestedAttribute{
									MarkdownDescription: "Parent building blocks this block depends on, forming a dependency hierarchy " +
										"in which a parent's outputs can feed this block's inputs (see [building block concepts](https://docs.meshcloud.io/concepts/building-block/)).",
									Computed: true,
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"buildingblock_uuid": computedString("UUID of the parent building block."),
											"definition_uuid":    computedString("UUID of the parent building block definition."),
										},
									},
								},
							},
						},
						"status": schema.SingleNestedAttribute{
							MarkdownDescription: "Current building block status.",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"status": computedString("Execution status. One of " + client.BuildingBlockStatuses.Markdown() + "."),
								"force_purge": schema.BoolAttribute{MarkdownDescription: "True once a purge has been requested for this building block. " +
									"A purge removes the block without a destroy run, leaving its cloud resources unmanaged (the lifecycle still reaches DELETED).", Computed: true},
								"latest_run_uuid":     computedString("UUID of the latest modifying (apply/destroy) run. Null when none exists or when permissions are insufficient to read runs."),
								"latest_dry_run_uuid": computedString("UUID of the latest dry (DETECT) run, but only when it is the newest run; null otherwise."),
								"outputs": schema.MapNestedAttribute{
									MarkdownDescription: "Outputs of the building block, available after a successful run.",
									Computed:            true,
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"value": schema.StringAttribute{
												CustomType:          jsontypes.NormalizedType{},
												MarkdownDescription: "Output value. Use `jsondecode(...)` to obtain a polymorphic value depending on `value_type`.",
												Computed:            true,
											},
											"value_type":      computedString("Data type of the value. One of " + client.MeshBuildingBlockIOTypes.Markdown() + "."),
											"assignment_type": computedString("How the output value is assigned. One of " + client.MeshBuildingBlockDefinitionOutputAssignmentTypes.Markdown() + "."),
										},
									},
								},
							},
						},
						"all_inputs": schema.MapNestedAttribute{
							MarkdownDescription: "View of **all** inputs resolved by the backend — platform-operator, user, and " +
								"static inputs (the latter derived from the BBD) — regardless of who set them.<br>" +
								"Non-sensitive inputs show their plain value; sensitive inputs show only their hash.",
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"value": computedString("Non-sensitive input value, as a `jsonencode`d representation (e.g. `\"my-name\"` or `16`)."),
									"sensitive": secret.DatasourceSchema(secret.DatasourceSchemaOptions{
										MarkdownDescription: "Sensitive input value (hash only).",
									}),
									"value_type":      computedString("Data type of the value. One of " + client.MeshBuildingBlockIOTypes.Markdown() + "."),
									"assignment_type": computedString("How the input value is assigned."),
								},
							},
						},
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

	filter := client.MeshBuildingBlockV2ListFilter{
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
	}

	blocks, err := d.client.List(ctx, &filter)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list building blocks", err.Error())
		return
	}

	config.BuildingBlocks = make([]buildingBlockListItem, 0, len(blocks))
	for i := range blocks {
		bb := blocks[i]
		item := buildingBlockListItem{
			Metadata: bb.Metadata,
			Spec: buildingBlockListItemSpec{
				DisplayName:                       bb.Spec.DisplayName,
				BuildingBlockDefinitionVersionRef: buildingBlockListItemVersionRef{Uuid: bb.Spec.BuildingBlockDefinitionVersionRef.Uuid},
				TargetRef:                         bb.Spec.TargetRef,
				ParentBuildingBlocks:              bb.Spec.ParentBuildingBlocks,
			},
			Status:    bb.Status,
			AllInputs: make(map[string]buildingBlockAllInput, len(bb.Spec.Inputs)),
		}
		for key, input := range bb.Spec.Inputs {
			item.AllInputs[key] = buildAllInput(input, &resp.Diagnostics)
		}
		config.BuildingBlocks = append(config.BuildingBlocks, item)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// generic.Set handles the client-specific types in the item models: clientTypes.Any (outputs
	// value) and clientTypes.Set (parent_building_blocks). all_inputs sensitive values are already
	// reduced to a hash-only secret.HashOnly by buildAllInput, so no secret converter is needed.
	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, config,
		withValueFromConverterForClientTypeAny(),
		generic.WithSliceTypeAsSet(clientTypes.IsSet),
	)...)
}
