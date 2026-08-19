package provider

import (
	"context"
	"fmt"
	"maps"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

// buildingBlockV0Parent is the flat parent element that pre-v1 state carries. The upgrader declares
// it instead of reusing a client DTO, because the state shape outlives the v1 API whose DTO once
// matched it.
type buildingBlockV0Parent struct {
	BuildingBlockUuid string `tfsdk:"buildingblock_uuid"`
	DefinitionUuid    string `tfsdk:"definition_uuid"`
}

// buildingBlockSchemaV0Once derives the prior schema from the current one. That is safe because v0
// and v1 state differ in only two attributes: v0 held the parents in spec.parent_building_blocks as
// a flat {buildingblock_uuid, definition_uuid} pair, which v1 renames to
// spec.parent_building_block_refs and reshapes into refs; and `ref` did not exist yet.
var buildingBlockSchemaV0Once = sync.OnceValue(func() schema.Schema {
	var schemaResp resource.SchemaResponse
	(&buildingBlockResource{}).Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema
	s.Version = 0

	spec, ok := s.Attributes["spec"].(schema.SingleNestedAttribute)
	if !ok {
		panic("building block spec attribute is not a SingleNestedAttribute")
	}
	spec.Attributes = maps.Clone(spec.Attributes)
	delete(spec.Attributes, "parent_building_block_refs")
	spec.Attributes["parent_building_blocks"] = schema.SetNestedAttribute{
		Optional: true,
		Computed: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"buildingblock_uuid": schema.StringAttribute{Required: true},
				"definition_uuid":    schema.StringAttribute{Required: true},
			},
		},
	}

	s.Attributes = maps.Clone(s.Attributes)
	s.Attributes["spec"] = spec
	// `ref` did not exist at v0; declaring it would decode as null instead of being recomputed.
	delete(s.Attributes, "ref")

	return s
})

func (r *buildingBlockResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	prior := buildingBlockSchemaV0Once()
	return map[int64]resource.StateUpgrader{
		0: {PriorSchema: &prior, StateUpgrader: r.upgradeParentsToRefs},
	}
}

// upgradeParentsToRefs rewrites v0 state onto the v1 schema: the flat parent pair becomes a set of
// refs and the new computed `ref` is derived from metadata.uuid. Everything else carries over
// untouched, so the raw prior value is copied attribute by attribute rather than round-tripped
// through the resource model — the model's inputs need the plan/config-aware secret converters,
// which an upgrader has no access to.
//
// Without this upgrader, spec.parent_building_block_refs stays null after the upgrade. Under
// -refresh=false that looks like a parent change on an unchanged version, and
// requiresReplaceParentsWhenVersionUnchanged then destroys and recreates a live building block.
func (r *buildingBlockResource) upgradeParentsToRefs(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	var flatParents []buildingBlockV0Parent
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("spec").AtName("parent_building_blocks"), &flatParents)...)

	var uuid string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)
	if resp.Diagnostics.HasError() {
		return
	}

	schemaType, ok := resp.State.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		resp.Diagnostics.AddError("Unexpected building block schema type", "the resource schema is not an object type")
		return
	}
	specType, ok := schemaType.AttributeTypes["spec"].(tftypes.Object)
	if !ok {
		resp.Diagnostics.AddError("Unexpected building block schema type", "spec is not an object type")
		return
	}
	refsType, ok := specType.AttributeTypes["parent_building_block_refs"].(tftypes.Set)
	if !ok {
		resp.Diagnostics.AddError("Unexpected building block schema type", "spec.parent_building_block_refs is not a set type")
		return
	}

	var attributes, spec map[string]tftypes.Value
	if err := req.State.Raw.As(&attributes); err != nil {
		resp.Diagnostics.AddError("Could not read prior building block state", err.Error())
		return
	}
	if err := attributes["spec"].As(&spec); err != nil {
		resp.Diagnostics.AddError("Could not read prior building block spec", err.Error())
		return
	}

	// The parent's definition uuid is dropped: meshStack derives it from the referenced block.
	refs := make([]tftypes.Value, 0, len(flatParents))
	for _, parent := range flatParents {
		refs = append(refs, tftypes.NewValue(refsType.ElementType, map[string]tftypes.Value{
			"kind": tftypes.NewValue(tftypes.String, client.MeshObjectKind.BuildingBlock),
			"uuid": tftypes.NewValue(tftypes.String, parent.BuildingBlockUuid),
		}))
	}

	delete(spec, "parent_building_blocks")
	spec["parent_building_block_refs"] = tftypes.NewValue(refsType, refs)
	attributes["spec"] = tftypes.NewValue(specType, spec)
	attributes["ref"] = tftypes.NewValue(schemaType.AttributeTypes["ref"], map[string]tftypes.Value{
		"kind": tftypes.NewValue(tftypes.String, client.MeshObjectKind.BuildingBlock),
		"uuid": tftypes.NewValue(tftypes.String, uuid),
	})

	if err := tftypes.ValidateValue(schemaType, attributes); err != nil {
		resp.Diagnostics.AddError("Could not upgrade building block state", fmt.Sprintf("upgraded state does not match the current schema: %s", err))
		return
	}
	resp.State.Raw = tftypes.NewValue(schemaType, attributes)
}
