package generic

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// targetRefType mirrors a meshObject ref block, the shallowest place a planned spec carries an
// unknown that ValueTo cannot represent.
var targetRefType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"kind": tftypes.String,
	"uuid": tftypes.String,
}}

var specType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"display_name": tftypes.String,
	"target_ref":   targetRefType,
}}

var planType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{"spec": specType}}

func planWithTargetRefUuid(uuid tftypes.Value) tftypes.Value {
	return tftypes.NewValue(planType, map[string]tftypes.Value{
		"spec": tftypes.NewValue(specType, map[string]tftypes.Value{
			"display_name": tftypes.NewValue(tftypes.String, "Forgejo Connector Dev"),
			"target_ref": tftypes.NewValue(targetRefType, map[string]tftypes.Value{
				"kind": tftypes.NewValue(tftypes.String, "meshTenant"),
				"uuid": uuid,
			}),
		}),
	})
}

func TestAttributeHasUnknown(t *testing.T) {
	tests := []struct {
		name string
		plan tftypes.Value
		want bool
	}{
		{
			name: "fully known",
			plan: planWithTargetRefUuid(tftypes.NewValue(tftypes.String, "124b09ec-63b8-452e-a837-44afb382d5bd")),
			want: false,
		},
		{
			// A tenant the plan replaces has an unknown uuid, which reaches target_ref.uuid through
			// for_each. The enclosing objects stay known, so only a walk finds it.
			name: "unknown nested two levels deep",
			plan: planWithTargetRefUuid(tftypes.NewValue(tftypes.String, tftypes.UnknownValue)),
			want: true,
		},
		{
			name: "null counts as known",
			plan: planWithTargetRefUuid(tftypes.NewValue(tftypes.String, nil)),
			want: false,
		},
		{
			name: "the whole attribute is unknown",
			plan: tftypes.NewValue(planType, map[string]tftypes.Value{
				"spec": tftypes.NewValue(specType, tftypes.UnknownValue),
			}),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AttributeHasUnknown(tt.plan, "spec")
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAttributeHasUnknown_MissingAttribute(t *testing.T) {
	noSpec := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"other": tftypes.String}}
	plan := tftypes.NewValue(noSpec, map[string]tftypes.Value{
		"other": tftypes.NewValue(tftypes.String, "value"),
	})

	_, err := AttributeHasUnknown(plan, "spec")
	require.Error(t, err, "a missing attribute is a programming error and must not read as known")
}
