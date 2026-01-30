package generic

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testCaseForType[T Supported] struct {
	Want      attr.Type
	WantPanic assert.PanicAssertionFunc

	Values      []any
	WantValues  []string
	ValuesPanic []any

	ValueAssertEqual assert.ComparisonAssertionFunc
}

func (tt testCaseForType[T]) Run(t *testing.T) {
	t.Helper()
	var zero T
	t.Run(fmt.Sprintf("%T", zero), func(t *testing.T) {
		if tt.WantPanic != nil {
			tt.WantPanic(t, func() { _ = TypeFor[T]() })
			return
		}
		sut := TypeFor[T]()
		nullType := sut.underlyingNull.Type(context.Background())
		assert.Truef(t, sut.underlyingType.Equal(nullType), "Underlying type %s does not match type of null %s", sut.underlyingType, nullType)
		assert.Truef(t, sut.underlyingType.Equal(tt.Want), "Underlying type %s does not match expected type %s", sut.underlyingType, tt.Want)
		require.Len(t, tt.WantValues, len(tt.Values))
		for i, testValue := range tt.Values {
			t.Run(fmt.Sprintf("value %#v", testValue), func(t *testing.T) {
				var diags diag.Diagnostics
				v := sut.valueFactory(testValue, &diags)
				require.Empty(t, diags)

				t.Run("type/string", func(t *testing.T) {
					vType := v.Type(context.Background())
					assert.Truef(t, sut.Equal(vType), "Value type %s does not match expected type %s", vType, sut)
					type ValueStringer interface {
						ValueString() string
					}
					if valueStringer, ok := v.Value.(ValueStringer); ok {
						assert.Equal(t, tt.WantValues[i], valueStringer.ValueString())
					} else {
						assert.Equal(t, tt.WantValues[i], v.String())
					}
				})

				if !v.IsNull() {
					t.Run("round-trip", func(t *testing.T) {
						var diags diag.Diagnostics
						vAgain := v.Get(&diags)
						require.Empty(t, diags)
						if tt.ValueAssertEqual != nil {
							tt.ValueAssertEqual(t, testValue, vAgain)
						} else {
							assert.Equal(t, testValue, vAgain)
						}
					})
				} else {
					// only relevant for T=any
					t.Run("round-trip null", func(t *testing.T) {
						var diags diag.Diagnostics
						vAgain := v.Get(&diags)
						require.Empty(t, diags)
						assert.Nil(t, vAgain)
					})
				}
			})
		}

		for _, testValuePanic := range tt.ValuesPanic {
			t.Run(fmt.Sprintf("panic value %#v", testValuePanic), func(t *testing.T) {
				assert.Panics(t, func() {
					var diags diag.Diagnostics
					_ = sut.valueFactory(testValuePanic, &diags)
				})
			})
		}
	})
}

func TestTypeFor(t *testing.T) {
	type (
		stringLike string
		int64Like  int64
	)

	testCaseForType[string]{
		Want:        basetypes.StringType{},
		Values:      []any{"1", "", "some string"},
		WantValues:  []string{`1`, ``, `some string`},
		ValuesPanic: []any{7, true, nil},
	}.Run(t)
	testCaseForType[stringLike]{Want: basetypes.StringType{}}.Run(t)

	const (
		MaxUint64 = ^uint64(0)
		MaxInt64  = int64(MaxUint64 >> 1)
	)
	testCaseForType[int64]{
		Want:        basetypes.Int64Type{},
		Values:      []any{1, 2, 0, MaxInt64},
		WantValues:  []string{`1`, `2`, `0`, `9223372036854775807`},
		ValuesPanic: []any{"string", true},
		ValueAssertEqual: func(t assert.TestingT, expected any, actual any, msgAndArgs ...any) bool {
			return assert.InDelta(t, expected, actual, 0, msgAndArgs...)
		},
	}.Run(t)
	testCaseForType[int64Like]{Want: basetypes.Int64Type{}}.Run(t)

	var nothing any
	someString := "pointer string"
	testCaseForType[any]{
		Want:       jsontypes.NormalizedType{},
		Values:     []any{true, 8, false, MaxInt64, "some string", nil, &nothing, "😍", &someString, float32(8.1231)},
		WantValues: []string{`true`, `8`, `false`, `9223372036854775807`, `"some string"`, ``, ``, `"😍"`, `"pointer string"`, `8.1231`},
		ValueAssertEqual: func(t assert.TestingT, expected any, actual any, msgAndArgs ...any) bool {
			if _, isFloat32 := expected.(float32); isFloat32 {
				return assert.InDelta(t, expected, actual, 1e-6, msgAndArgs...)
			} else if _, isNumeric := actual.(float64); isNumeric {
				return assert.InDelta(t, expected, actual, 0, msgAndArgs...)
			} else {
				if stringPtr, ok := expected.(*string); ok {
					expected = *stringPtr
				}
				return assert.Equal(t, expected, actual, msgAndArgs...)
			}
		},
	}.Run(t)

	// Currently unsupported types:
	testCaseForType[bool]{WantPanic: assert.Panics}.Run(t)
	testCaseForType[struct{}]{WantPanic: assert.Panics}.Run(t)
}
