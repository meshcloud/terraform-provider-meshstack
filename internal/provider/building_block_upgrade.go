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

// parentRefsFromFlatParents drops the parent's definition uuid, because meshStack derives it from
// the referenced building block.
func parentRefsFromFlatParents(parents []client.MeshBuildingBlockParent) []client.UuidRef {
	refs := make([]client.UuidRef, 0, len(parents))
	for _, parent := range parents {
		refs = append(refs, client.UuidRef{Kind: client.MeshObjectKind.BuildingBlock, Uuid: parent.BuildingBlockUuid})
	}
	return refs
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
	delete(s.Attributes, "ref")

	return s
})

func (r *buildingBlockResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	prior := buildingBlockSchemaV0Once()
	return map[int64]resource.StateUpgrader{
		0: {PriorSchema: &prior, StateUpgrader: r.upgradeParentsToRefs},
	}
}

// upgradeParentsToRefs copies the prior state attribute by attribute instead of round-tripping it
// through the resource model, because reading the model back needs the plan/config-aware secret
// converters that an upgrader has no access to.
//
// Without this upgrader, spec.parent_building_block_refs stays null after the upgrade. Under
// -refresh=false that looks like a parent change on an unchanged version, and
// requiresReplaceParentsWhenVersionUnchanged then destroys and recreates a live building block.
func (r *buildingBlockResource) upgradeParentsToRefs(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	var flatParents []client.MeshBuildingBlockParent
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

	refs := make([]tftypes.Value, 0, len(flatParents))
	for _, ref := range parentRefsFromFlatParents(flatParents) {
		refs = append(refs, tftypes.NewValue(refsType.ElementType, map[string]tftypes.Value{
			"kind": tftypes.NewValue(tftypes.String, ref.Kind),
			"uuid": tftypes.NewValue(tftypes.String, ref.Uuid),
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
