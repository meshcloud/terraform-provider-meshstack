package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/secret"
)

var (
	_ datasource.DataSource              = &buildingBlockDataSource{}
	_ datasource.DataSourceWithConfigure = &buildingBlockDataSource{}
)

func NewBuildingBlockDataSource() datasource.DataSource {
	return &buildingBlockDataSource{}
}

type buildingBlockDataSource struct {
	client client.MeshBuildingBlockV2Client
}

type buildingBlockDataSourceModel struct {
	Ref       client.UuidRef                     `tfsdk:"ref"`
	Metadata  client.MeshBuildingBlockV2Metadata `tfsdk:"metadata"`
	Spec      buildingBlockReadSpec              `tfsdk:"spec"`
	Status    *client.MeshBuildingBlockV2Status  `tfsdk:"status"`
	AllInputs map[string]buildingBlockAllInput   `tfsdk:"all_inputs"`
}

func (d *buildingBlockDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_building_block"
}

func (d *buildingBlockDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.client = client.BuildingBlockV2
	})...)
}

func (d *buildingBlockDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	status := buildingBlockReadStatusAttribute()

	resp.Schema = schema.Schema{
		MarkdownDescription: "Read a single building block by UUID. " +
			"It is read-only and mirrors the `meshstack_building_block` resource " +
			"(`metadata`/`spec`/`status`/`all_inputs`, plus the resource's computed `ref`). " +
			"Use `meshstack_building_blocks` to look building blocks up by filter instead of by UUID." +
			previewDisclaimer(),
		Attributes: map[string]schema.Attribute{
			"ref": meshRefByUuid(meshRefOptions{Kind: client.MeshObjectKind.BuildingBlock, Description: "Reference to this building block, can be used in another building block's `spec.parent_building_block_refs`.", Output: true}),

			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Building block metadata.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"uuid": schema.StringAttribute{
						MarkdownDescription: "UUID which uniquely identifies the building block.",
						Required:            true,
					},
					"owned_by_workspace": computedString("The workspace containing this building block."),
				},
			},

			"spec":       buildingBlockReadSpecAttribute(),
			"status":     status,
			"all_inputs": buildingBlockReadAllInputsAttribute(),
		},
	}
}

func (d *buildingBlockDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var uuid string
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bb, err := d.client.Read(ctx, uuid)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read building block", err.Error())
		return
	}
	if bb == nil {
		resp.Diagnostics.AddError(
			"Building block not found",
			fmt.Sprintf("Building block with UUID '%s' was not found", uuid),
		)
		return
	}

	model := buildingBlockDataSourceModel{
		Ref:       client.UuidRef{Kind: client.MeshObjectKind.BuildingBlock, Uuid: *bb.Metadata.Uuid},
		Metadata:  bb.Metadata,
		Spec:      buildingBlockReadSpecFromDto(bb.Spec),
		Status:    bb.Status,
		AllInputs: buildingBlockAllInputsFromDto(bb.Spec.Inputs, &resp.Diagnostics),
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// buildAllInput has already reduced every sensitive all_inputs value to a secret.HashOnly, so no
	// secret converter is needed here.
	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, model,
		withValueFromConverterForClientTypeAny(),
		generic.WithSliceTypeAsSet(clientTypes.IsSet),
	)...)
}

func computedString(md string) schema.StringAttribute {
	return schema.StringAttribute{MarkdownDescription: md, Computed: true}
}

type buildingBlockReadSpec struct {
	DisplayName                       string                              `tfsdk:"display_name"`
	BuildingBlockDefinitionVersionRef buildingBlockReadVersionRef         `tfsdk:"building_block_definition_version_ref"`
	TargetRef                         client.MeshBuildingBlockV2TargetRef `tfsdk:"target_ref"`
	ParentBuildingBlockRefs           clientTypes.Set[client.UuidRef]     `tfsdk:"parent_building_block_refs"`
}

// buildingBlockReadVersionRef has no content_hash: that is a Terraform-only field which the backend
// never returns, so a read has nothing to fill it from.
type buildingBlockReadVersionRef struct {
	Uuid string `tfsdk:"uuid"`
	Kind string `tfsdk:"kind"`
}

func buildingBlockReadSpecFromDto(spec client.MeshBuildingBlockV2Spec) buildingBlockReadSpec {
	return buildingBlockReadSpec{
		DisplayName: spec.DisplayName,
		BuildingBlockDefinitionVersionRef: buildingBlockReadVersionRef{
			Uuid: spec.BuildingBlockDefinitionVersionRef.Uuid,
			Kind: client.MeshObjectKind.BuildingBlockDefinitionVersion,
		},
		TargetRef:               spec.TargetRef,
		ParentBuildingBlockRefs: spec.ParentBuildingBlockRefs,
	}
}

func buildingBlockAllInputsFromDto(inputs map[string]*client.MeshBuildingBlockInput, diags *diag.Diagnostics) map[string]buildingBlockAllInput {
	allInputs := make(map[string]buildingBlockAllInput, len(inputs))
	for key, input := range inputs {
		allInputs[key] = buildAllInput(input, diags)
	}
	return allInputs
}

func buildingBlockReadSpecAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Building block specification.",
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			"display_name": computedString("Display name for the building block as shown in meshPanel."),
			"building_block_definition_version_ref": schema.SingleNestedAttribute{
				MarkdownDescription: "References the building block definition version this building block is based on.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"uuid": computedString("UUID of the building block definition version."),
					"kind": computedString("meshObject type, always `" + client.MeshObjectKind.BuildingBlockDefinitionVersion + "`."),
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
			// meshRefByUuid cannot build this {kind, uuid} element: it returns a resource schema
			// attribute, whose Attributes map does not fit a data source NestedAttributeObject.
			"parent_building_block_refs": schema.SetNestedAttribute{
				MarkdownDescription: "Set of refs to the parent building blocks this block depends on, forming a dependency hierarchy " +
					"in which a parent's outputs can feed this block's inputs (see [building block concepts](https://docs.meshcloud.io/concepts/building-block/)).",
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"kind": computedString("meshObject type, always `" + client.MeshObjectKind.BuildingBlock + "`."),
						"uuid": computedString("UUID (`metadata.uuid`) of the parent `" + client.MeshObjectKind.BuildingBlock + "`."),
					},
				},
			},
		},
	}
}

func buildingBlockReadStatusAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
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
	}
}

func buildingBlockReadAllInputsAttribute() schema.MapNestedAttribute {
	return schema.MapNestedAttribute{
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
	}
}
