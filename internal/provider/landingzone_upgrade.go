package provider

import (
	"context"
	"maps"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/meshcloud/meshstack-cli/client"

	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
)

// required for backwards compatibility of older versions of LZ in meshObject API.
type landingZoneModelV0 struct {
	Metadata client.MeshLandingZoneMetadata `tfsdk:"metadata"`
	Spec     client.MeshLandingZoneSpec     `tfsdk:"spec"`
	Status   client.MeshLandingZoneStatus   `tfsdk:"status"`
}

// landingZoneSchemaV0Once builds the legacy schema for the state upgrader:
// the current schema with metadata.tags as its old set(string) type.
// v1 corrected tags to list(string) so this exists only to read legacy state.
var landingZoneSchemaV0Once = sync.OnceValue(func() schema.Schema {
	var schemaResp resource.SchemaResponse
	(&landingZoneResource{}).Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema
	s.Version = 0 // indicator for legacy version

	metadata, ok := s.Attributes["metadata"].(schema.SingleNestedAttribute)
	if !ok {
		panic("landing zone metadata attribute is not a SingleNestedAttribute")
	}
	metadata.Attributes = maps.Clone(metadata.Attributes)
	metadata.Attributes["tags"] = schema.MapAttribute{
		ElementType: types.SetType{ElemType: types.StringType},
		Optional:    true,
		Computed:    true,
		Default:     mapdefault.StaticValue(types.MapValueMust(types.SetType{ElemType: types.StringType}, map[string]attr.Value{})),
	}
	s.Attributes = maps.Clone(s.Attributes)
	s.Attributes["metadata"] = metadata

	// important to drop here to not have unexpected null values (the field did not exist at all in legacy LZ)
	delete(s.Attributes, "ref")

	return s
})

// upgradeTagsSetToList migrates legacy tags to proper LZ v1 state.
func (r *landingZoneResource) upgradeTagsSetToList(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	// generic.Get rather than req.State.Get: the prior schema is derived from the live one, so every
	// attribute added since v0 decodes as null out of legacy state. The framework refuses to put a
	// null into a value-typed field, generic.Get leaves it at its zero value — which is the schema
	// default for those attributes anyway, and the refresh right after replaces it with the API's.
	prior := generic.Get[landingZoneModelV0](ctx, req.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, landingZoneModel{
		Ref:      landingZoneRefOutput{Name: prior.Metadata.Name, Kind: client.MeshObjectKind.LandingZone},
		Metadata: prior.Metadata,
		Spec:     prior.Spec,
		Status:   prior.Status,
	})...)
}
