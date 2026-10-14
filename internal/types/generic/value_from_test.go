package generic

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"
)

type priorValues map[string]attr.Value

func (p priorValues) Get(context.Context, any) diag.Diagnostics {
	panic("not needed")
}

func (p priorValues) GetAttribute(_ context.Context, attributePath path.Path, target any) (diags diag.Diagnostics) {
	value, ok := p[attributePath.String()]
	if !ok {
		diags.AddAttributeError(attributePath, "no prior value", "")
		return
	}
	if attributeValue, ok := target.(*attr.Value); ok {
		*attributeValue = value
	}
	return
}

func TestWithEmptyContainersAsIn(t *testing.T) {
	type model struct {
		Items map[string]string `tfsdk:"items"`
		Other []string          `tfsdk:"other"`
	}
	itemsPath := path.Root("items")
	itemsType := tftypes.Map{ElementType: tftypes.String}
	null := tftypes.NewValue(itemsType, nil)
	empty := tftypes.NewValue(itemsType, map[string]tftypes.Value{})

	itemsFrom := func(t *testing.T, in model, prior AttributeGetter) tftypes.Value {
		t.Helper()
		out, err := ValueFrom(in, WithEmptyContainersAsIn(t.Context(), prior, itemsPath))
		require.NoError(t, err)
		var attributes map[string]tftypes.Value
		require.NoError(t, out.As(&attributes))
		require.Equal(t, tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{}), attributes["other"],
			"an attribute that is not listed keeps its empty value")
		return attributes["items"]
	}
	priorNull := priorValues{itemsPath.String(): types.MapNull(types.StringType)}
	priorEmpty := priorValues{itemsPath.String(): types.MapValueMust(types.StringType, map[string]attr.Value{})}

	t.Run("prior null gives null", func(t *testing.T) {
		require.Equal(t, null, itemsFrom(t, model{Items: map[string]string{}, Other: []string{}}, priorNull))
	})
	t.Run("prior empty gives empty, also for a nil map", func(t *testing.T) {
		require.Equal(t, empty, itemsFrom(t, model{Items: map[string]string{}, Other: []string{}}, priorEmpty))
		require.Equal(t, empty, itemsFrom(t, model{Other: []string{}}, priorEmpty))
	})
	t.Run("no prior value gives null", func(t *testing.T) {
		require.Equal(t, null, itemsFrom(t, model{Items: map[string]string{}, Other: []string{}}, priorValues{}))
		require.Equal(t, null, itemsFrom(t, model{Items: map[string]string{}, Other: []string{}}, nil))
	})
	t.Run("a non-empty value ignores the prior", func(t *testing.T) {
		require.Equal(t,
			tftypes.NewValue(itemsType, map[string]tftypes.Value{"a": tftypes.NewValue(tftypes.String, "b")}),
			itemsFrom(t, model{Items: map[string]string{"a": "b"}, Other: []string{}}, priorNull))
	})
}
