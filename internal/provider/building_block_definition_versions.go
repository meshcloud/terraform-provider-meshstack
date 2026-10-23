package provider

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"math/big"
	"slices"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const versionsKeyDescription = "Keyed by `v<number>`, such as `versions[\"v2\"]`. " +
	"A `for` expression visits the keys in lexical order, so `v10` comes before `v2`."

func versionKey(number int64) string {
	return fmt.Sprintf("v%d", number)
}

func versionsByKey[V any](ascending []V, number func(V) int64) map[string]V {
	byKey := make(map[string]V, len(ascending))
	for _, version := range ascending {
		byKey[versionKey(number(version))] = version
	}
	return byKey
}

func (ref buildingBlockDefinitionVersionRef) number() int64 {
	return ref.Number.Get()
}

func ascendingVersions(byKey map[string]buildingBlockDefinitionVersionRef) []buildingBlockDefinitionVersionRef {
	return slices.SortedFunc(maps.Values(byKey), func(a, b buildingBlockDefinitionVersionRef) int {
		return cmp.Compare(a.number(), b.number())
	})
}

var buildingBlockDefinitionSchemaV0Once = sync.OnceValue(func() schema.Schema {
	var schemaResp resource.SchemaResponse
	(&buildingBlockDefinitionResource{}).Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema
	s.Version = 0

	versions, ok := s.Attributes["versions"].(schema.MapNestedAttribute)
	if !ok {
		panic("building block definition versions attribute is not a MapNestedAttribute")
	}
	s.Attributes = maps.Clone(s.Attributes)
	s.Attributes["versions"] = schema.ListNestedAttribute{Computed: true, NestedObject: versions.NestedObject}
	return s
})

func (r *buildingBlockDefinitionResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	prior := buildingBlockDefinitionSchemaV0Once()
	return map[int64]resource.StateUpgrader{
		0: {PriorSchema: &prior, StateUpgrader: r.upgradeVersionsListToMap},
	}
}

// upgradeVersionsListToMap rewrites the raw state, because reading it into the model needs the
// plan- and config-aware secret converters, which an upgrader has no access to.
func (r *buildingBlockDefinitionResource) upgradeVersionsListToMap(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	var attributes map[string]tftypes.Value
	if err := req.State.Raw.As(&attributes); err != nil {
		resp.Diagnostics.AddError("Cannot read the prior building block definition state", err.Error())
		return
	}
	versionsType, typeDiags := resp.State.Schema.TypeAtPath(ctx, path.Root("versions"))
	resp.Diagnostics.Append(typeDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var ascending []tftypes.Value
	if err := attributes["versions"].As(&ascending); err != nil {
		resp.Diagnostics.AddError("Cannot read versions from the prior building block definition state", err.Error())
		return
	}
	const numberSummary = "Cannot read a version number from the prior building block definition state"
	byKey := make(map[string]tftypes.Value, len(ascending))
	for _, version := range ascending {
		var fields map[string]tftypes.Value
		var number *big.Float
		if err := version.As(&fields); err != nil {
			resp.Diagnostics.AddError("Cannot read a version from the prior building block definition state", err.Error())
			return
		}
		if err := fields["number"].As(&number); err != nil {
			resp.Diagnostics.AddError(numberSummary, err.Error())
			return
		}
		if number == nil {
			resp.Diagnostics.AddError(numberSummary, "A version has no number.")
			return
		}
		numberInt, accuracy := number.Int64()
		if accuracy != big.Exact {
			resp.Diagnostics.AddError(numberSummary, fmt.Sprintf("Version number %v is not a whole number.", number))
			return
		}
		key := versionKey(numberInt)
		if _, taken := byKey[key]; taken {
			resp.Diagnostics.AddError(numberSummary, fmt.Sprintf("Version number %d occurs twice.", numberInt))
			return
		}
		byKey[key] = version
	}
	if attributes["versions"].IsNull() {
		byKey = nil
	}
	attributes["versions"] = tftypes.NewValue(versionsType.TerraformType(ctx), byKey)
	resp.State.Raw = tftypes.NewValue(resp.State.Schema.Type().TerraformType(ctx), attributes)
}
