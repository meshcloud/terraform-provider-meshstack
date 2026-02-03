package generic

import (
	"context"
	"fmt"
	"maps"
	"math"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testCaseForType[T any] struct {
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

	t.Run("string", func(t *testing.T) {
		testCaseForType[string]{
			Want:        basetypes.StringType{},
			Values:      []any{"1", "", "some string"},
			WantValues:  []string{`1`, ``, `some string`},
			ValuesPanic: []any{7, true, nil},
		}.Run(t)
		type StringLike string
		testCaseForType[StringLike]{Want: basetypes.StringType{}}.Run(t)
	})

	t.Run("bool", func(t *testing.T) {
		testCaseForType[bool]{
			Want:        basetypes.BoolType{},
			Values:      []any{true, false},
			WantValues:  []string{`true`, `false`},
			ValuesPanic: []any{"", 7, nil},
		}.Run(t)
		type BoolLike bool
		testCaseForType[BoolLike]{Want: basetypes.BoolType{}}.Run(t)
	})

	t.Run("int64", func(t *testing.T) {
		testCaseForType[int64]{
			Want:        basetypes.Int64Type{},
			Values:      []any{1, 2, 0, math.MaxInt64},
			WantValues:  []string{`1`, `2`, `0`, `9223372036854775807`},
			ValuesPanic: []any{"string", true},
			ValueAssertEqual: func(t assert.TestingT, expected any, actual any, msgAndArgs ...any) bool {
				return assert.InDelta(t, expected, actual, 0, msgAndArgs...)
			},
		}.Run(t)
		type Int64Like int64
		testCaseForType[Int64Like]{Want: basetypes.Int64Type{}}.Run(t)
	})

	t.Run("any", func(t *testing.T) {
		var nothing any
		someString := "pointer string"
		testCaseForType[any]{
			Want:       jsontypes.NormalizedType{},
			Values:     []any{true, 8, false, math.MaxInt64, "some string", nil, &nothing, "😍", &someString, float32(8.1231)},
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
	})

	t.Run("struct", func(t *testing.T) {
		type (
			EmptyStruct  struct{}
			SimpleStruct struct {
				A       string `tfsdk:"a"`
				B       int64  `tfsdk:"b"`
				C       bool   `tfsdk:"c"`
				Ignored string `tfsdk:"-"`
			}
			NestedStruct struct {
				SimpleStruct
				More SimpleStruct `tfsdk:"more"`
			}
		)

		testCaseForType[EmptyStruct]{
			Want:       basetypes.ObjectType{},
			Values:     []any{EmptyStruct{}},
			WantValues: []string{"{}"},
		}.Run(t)

		simpleAttributeTypes := map[string]attr.Type{
			"a": TypeFor[string](),
			"b": TypeFor[int64](),
			"c": TypeFor[bool](),
		}
		testCaseForType[SimpleStruct]{
			Want:       basetypes.ObjectType{AttrTypes: simpleAttributeTypes},
			Values:     []any{SimpleStruct{}},
			WantValues: []string{`{"a":"","b":0,"c":false}`},
		}.Run(t)

		nestedAttributeTypes := maps.Clone(simpleAttributeTypes)
		nestedAttributeTypes["more"] = basetypes.ObjectType{AttrTypes: simpleAttributeTypes}
		testCaseForType[NestedStruct]{
			Want:       basetypes.ObjectType{AttrTypes: nestedAttributeTypes},
			Values:     []any{NestedStruct{}},
			WantValues: []string{`{"a":"","b":0,"c":false,"more":{"a":"","b":0,"c":false}}`},
		}.Run(t)
	})

	// Currently unsupported types:
	t.Run("unsupported", func(t *testing.T) {
		testCaseForType[int]{WantPanic: assert.Panics}.Run(t)
		testCaseForType[float32]{WantPanic: assert.Panics}.Run(t)
		testCaseForType[complex64]{WantPanic: assert.Panics}.Run(t)

		testCaseForType[struct {
			A complex64 `tfsdk:"a"` // nested 'complex64' field also unsupported
		}]{WantPanic: assert.Panics}.Run(t)
	})
}
