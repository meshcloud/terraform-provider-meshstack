package generic

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestValueToFromTestcase[T any] struct {
	Value   T
	TfValue tftypes.Value
}

func (tt TestValueToFromTestcase[T]) Run(t *testing.T) {
	t.Helper()
	t.Run("ValueTo", func(t *testing.T) {
		t.Run("non-pointer", func(t *testing.T) {
			out, err := ValueTo[T](tt.TfValue)
			require.NoError(t, err)
			require.Equal(t, tt.Value, out)
		})
		t.Run("pointer", func(t *testing.T) {
			out, err := ValueTo[*T](tt.TfValue)
			require.NoError(t, err)
			require.Equal(t, &tt.Value, out)
		})
	})

	t.Run("ValueFrom", func(t *testing.T) {
		t.Run("non-pointer", func(t *testing.T) {
			out, err := ValueFrom(tt.Value)
			require.NoError(t, err)
			require.Equal(t, tt.TfValue, out)
		})
		t.Run("pointer", func(t *testing.T) {
			out, err := ValueFrom(&tt.Value)
			require.NoError(t, err)
			require.Equal(t, tt.TfValue, out)
		})
	})
}

func TestValueToFrom(t *testing.T) {
	t.Run("simple object", func(t *testing.T) {
		type testStruct struct {
			A string `tfsdk:"a"`
			B bool   `tfsdk:"b"`
			C int64  `tfsdk:"c"`
		}
		TestValueToFromTestcase[testStruct]{
			Value: testStruct{A: "test", B: true, C: 42},
			TfValue: tftypes.NewValue(tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"a": tftypes.String,
					"b": tftypes.Bool,
					"c": tftypes.Number,
				},
			}, map[string]tftypes.Value{
				"a": tftypes.NewValue(tftypes.String, "test"),
				"b": tftypes.NewValue(tftypes.Bool, true),
				"c": tftypes.NewValue(tftypes.Number, 42),
			}),
		}.Run(t)
	})

	t.Run("list type", func(t *testing.T) {
		TestValueToFromTestcase[[]string]{
			Value: []string{"item1", "item2", "item3"},
			TfValue: tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, "item1"),
				tftypes.NewValue(tftypes.String, "item2"),
				tftypes.NewValue(tftypes.String, "item3"),
			}),
		}.Run(t)
	})

	t.Run("map type", func(t *testing.T) {
		TestValueToFromTestcase[map[string]string]{
			Value: map[string]string{"key1": "value1", "key2": "value2", "key3": "value3"},
			TfValue: tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{
				"key1": tftypes.NewValue(tftypes.String, "value1"),
				"key2": tftypes.NewValue(tftypes.String, "value2"),
				"key3": tftypes.NewValue(tftypes.String, "value3"),
			}),
		}.Run(t)
	})

	t.Run("nullablity", func(t *testing.T) {
		type (
			embeddedStruct struct {
				C string `tfsdk:"c"`
				D int64  `tfsdk:"d"`
			}
			nestedStruct struct {
				A string `tfsdk:"name"`
				B int64  `tfsdk:"value"`
			}
			testStruct struct {
				embeddedStruct
				A      *string       `tfsdk:"a"`
				B      bool          `tfsdk:"b"`
				E      []int64       `tfsdk:"e"`
				Nested *nestedStruct `tfsdk:"nested"`
				Empty  *struct{}     `tfsdk:"empty"`
			}
		)
		attrTypes := map[string]tftypes.Type{
			"c": tftypes.String,
			"d": tftypes.Number,
			"a": tftypes.String,
			"b": tftypes.Bool,
			"e": tftypes.List{ElementType: tftypes.Number},
			"nested": tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"name":  tftypes.String,
				"value": tftypes.Number,
			}},
			"empty": tftypes.Object{AttributeTypes: map[string]tftypes.Type{}},
		}
		t.Run("all non-nil", func(t *testing.T) {
			aValue := "test-value"
			TestValueToFromTestcase[testStruct]{
				Value: testStruct{
					embeddedStruct: embeddedStruct{C: "embedded-c", D: 99},
					A:              &aValue,
					B:              true,
					E:              []int64{},
					Nested:         &nestedStruct{A: "nested-name", B: 200},
					Empty:          &struct{}{},
				},
				TfValue: tftypes.NewValue(tftypes.Object{AttributeTypes: attrTypes}, map[string]tftypes.Value{
					"c": tftypes.NewValue(tftypes.String, "embedded-c"),
					"d": tftypes.NewValue(tftypes.Number, 99),
					"a": tftypes.NewValue(tftypes.String, "test-value"),
					"b": tftypes.NewValue(tftypes.Bool, true),
					"e": tftypes.NewValue(tftypes.List{ElementType: tftypes.Number}, []tftypes.Value{}),
					"nested": tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{
						"name":  tftypes.String,
						"value": tftypes.Number,
					}}, map[string]tftypes.Value{
						"name":  tftypes.NewValue(tftypes.String, "nested-name"),
						"value": tftypes.NewValue(tftypes.Number, 200),
					}),
					"empty": tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{}}, map[string]tftypes.Value{}),
				}),
			}.Run(t)
		})
		t.Run("all nil", func(t *testing.T) {
			TestValueToFromTestcase[testStruct]{
				Value: testStruct{
					embeddedStruct: embeddedStruct{C: "embedded-c", D: 99},
				},
				TfValue: tftypes.NewValue(tftypes.Object{AttributeTypes: attrTypes}, map[string]tftypes.Value{
					"c": tftypes.NewValue(tftypes.String, "embedded-c"),
					"d": tftypes.NewValue(tftypes.Number, 99),
					"a": tftypes.NewValue(tftypes.String, nil),
					"b": tftypes.NewValue(tftypes.Bool, false),
					"e": tftypes.NewValue(tftypes.List{ElementType: tftypes.Number}, nil),
					"nested": tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{
						"name":  tftypes.String,
						"value": tftypes.Number,
					}}, nil),
					"empty": tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{}}, nil),
				}),
			}.Run(t)
		})
	})

	t.Run("NullIsUnknown", func(t *testing.T) {
		type nestedStruct struct {
			N string `tfsdk:"n"`
		}
		type testStruct struct {
			A NullIsUnknown[string]        `tfsdk:"a"`
			B NullIsUnknown[nestedStruct]  `tfsdk:"b"`
			C NullIsUnknown[*nestedStruct] `tfsdk:"c"`
		}
		nestedType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"n": tftypes.String}}
		structType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"a": tftypes.String, "b": nestedType, "c": nestedType}}

		t.Run("all known", func(t *testing.T) {
			TestValueToFromTestcase[testStruct]{
				Value: testStruct{
					A: KnownValue("known1"),
					B: KnownValue(nestedStruct{N: "known2"}),
					C: KnownValue[*nestedStruct](nil),
				},
				TfValue: tftypes.NewValue(structType, map[string]tftypes.Value{
					"a": tftypes.NewValue(tftypes.String, "known1"),
					"b": tftypes.NewValue(nestedType, map[string]tftypes.Value{"n": tftypes.NewValue(tftypes.String, "known2")}),
					"c": tftypes.NewValue(nestedType, nil),
				}),
			}.Run(t)
		})

		t.Run("all unknown", func(t *testing.T) {
			TestValueToFromTestcase[testStruct]{
				Value: testStruct{},
				TfValue: tftypes.NewValue(structType, map[string]tftypes.Value{
					"a": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
					"b": tftypes.NewValue(nestedType, tftypes.UnknownValue),
					"c": tftypes.NewValue(nestedType, tftypes.UnknownValue),
				}),
			}.Run(t)
		})

	})

	t.Run("ValueTo Ptr-Ptr-Ptr", func(t *testing.T) {
		out, err := ValueTo[***string](tftypes.NewValue(tftypes.String, "value1"))
		require.NoError(t, err)
		require.Equal(t, ptrTo(ptrTo(ptrTo("value1"))), out)
	})

	t.Run("nested complex object", func(t *testing.T) {
		type nestedStruct struct {
			Name          string `tfsdk:"name"`
			Value         int64  `tfsdk:"value"`
			IgnoredField1 string `tfsdk:"-"`
		}
		type complexStruct struct {
			ID            string            `tfsdk:"id"`
			Tags          []string          `tfsdk:"tags"`
			Metadata      map[string]string `tfsdk:"metadata"`
			Nested        nestedStruct      `tfsdk:"nested"`
			IgnoredField2 int64             `tfsdk:"-"`
			IgnoredField3 bool              `tfsdk:"-"`
		}
		TestValueToFromTestcase[complexStruct]{
			Value: complexStruct{
				ID:       "test-id",
				Tags:     []string{"tag1", "tag2"},
				Metadata: map[string]string{"env": "prod", "team": "platform"},
				Nested:   nestedStruct{Name: "nested-name", Value: 100},
			},
			TfValue: tftypes.NewValue(tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"id":       tftypes.String,
					"tags":     tftypes.List{ElementType: tftypes.String},
					"metadata": tftypes.Map{ElementType: tftypes.String},
					"nested": tftypes.Object{
						AttributeTypes: map[string]tftypes.Type{
							"name":  tftypes.String,
							"value": tftypes.Number,
						},
					},
				},
			}, map[string]tftypes.Value{
				"id": tftypes.NewValue(tftypes.String, "test-id"),
				"tags": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
					tftypes.NewValue(tftypes.String, "tag1"),
					tftypes.NewValue(tftypes.String, "tag2"),
				}),
				"metadata": tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{
					"env":  tftypes.NewValue(tftypes.String, "prod"),
					"team": tftypes.NewValue(tftypes.String, "platform"),
				}),
				"nested": tftypes.NewValue(tftypes.Object{
					AttributeTypes: map[string]tftypes.Type{
						"name":  tftypes.String,
						"value": tftypes.Number,
					},
				}, map[string]tftypes.Value{
					"name":  tftypes.NewValue(tftypes.String, "nested-name"),
					"value": tftypes.NewValue(tftypes.Number, 100),
				}),
			}),
		}.Run(t)
	})

	t.Run("optional struct WithValueFromConverter", func(t *testing.T) {
		type optionalStruct struct {
			E int64 `tfsdk:"e"`
		}
		type nestedStruct struct {
			C bool `tfsdk:"c"`
		}
		type testStruct struct {
			A nestedStruct    `tfsdk:"a"`
			B *nestedStruct   `tfsdk:"b"`
			D *optionalStruct `tfsdk:"d"`
			F *int64          `tfsdk:"f"`
		}
		var convertedNestedStruct, convertedInt64 int
		out, err := ValueFrom(testStruct{F: ptrTo(int64(0))},
			WithValueFromConverterFor[nestedStruct](nil, func(attributePath path.Path, value nestedStruct) (tftypes.Value, error) {
				convertedNestedStruct++
				return ValueFrom(value)
			}),
			WithValueFromConverterFor[int64](nil, func(attributePath path.Path, value int64) (tftypes.Value, error) {
				convertedInt64++
				return ValueFrom(int64(42))
			}),
		)
		require.NoError(t, err)
		assert.Equal(t, 1, convertedNestedStruct)
		assert.Equal(t, 1, convertedInt64)

		nestedType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"c": tftypes.Bool}}
		optionalType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"e": tftypes.Number}}
		expectedValue := tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{"a": nestedType, "b": nestedType, "d": optionalType, "f": tftypes.Number}},
			map[string]tftypes.Value{
				"a": tftypes.NewValue(nestedType, map[string]tftypes.Value{"c": tftypes.NewValue(tftypes.Bool, false)}),
				"b": tftypes.NewValue(nestedType, nil),
				"d": tftypes.NewValue(optionalType, nil),
				"f": tftypes.NewValue(tftypes.Number, int64(42)),
			},
		)
		assert.Equal(t, expectedValue, out)
	})

	t.Run("struct with any field and WithValueToConverter", func(t *testing.T) {
		type nestedStruct struct {
			C bool `tfsdk:"c"`
		}
		type testStruct struct {
			A nestedStruct  `tfsdk:"a"`
			B *nestedStruct `tfsdk:"b"`
		}
		nestedType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"c": tftypes.Bool}}
		nestedValueFalse := map[string]tftypes.Value{"c": tftypes.NewValue(tftypes.Bool, false)}
		var converterCalled int
		out, err := ValueTo[testStruct](tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{"a": nestedType, "b": nestedType}},
			map[string]tftypes.Value{
				"a": tftypes.NewValue(nestedType, nestedValueFalse),
				"b": tftypes.NewValue(nestedType, nestedValueFalse),
			},
		), WithValueToConverterFor[nestedStruct](func(attributePath path.Path, in tftypes.Value) (nestedStruct, error) {
			// converter returns true (instead of false) to show that it has been called for all nestedStruct fields A, B
			converterCalled++
			return nestedStruct{true}, nil
		}))
		require.NoError(t, err)
		assert.Equal(t, 2, converterCalled)
		assert.Equal(t, testStruct{A: nestedStruct{true}, B: &nestedStruct{true}}, out)
	})

	t.Run("string-like WithValueToConverter", func(t *testing.T) {
		type stringLike string
		type testStruct struct {
			A []stringLike `tfsdk:"a"`
			B []string     `tfsdk:"b"`
			C stringLike   `tfsdk:"c"`
			D *stringLike  `tfsdk:"d"`
			E string       `tfsdk:"e"`
		}
		stringArrayType := tftypes.List{ElementType: tftypes.String}
		var called int
		out, err := ValueTo[testStruct](tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"a": stringArrayType, "b": stringArrayType,
				"c": tftypes.String, "d": tftypes.String, "e": tftypes.String,
			}},
			map[string]tftypes.Value{
				"a": tftypes.NewValue(stringArrayType, nil),
				"b": tftypes.NewValue(stringArrayType, nil),
				"c": tftypes.NewValue(tftypes.String, nil),
				"d": tftypes.NewValue(tftypes.String, nil),
				"e": tftypes.NewValue(tftypes.String, nil),
			},
		), WithValueToConverterFor[stringLike](func(attributePath path.Path, in tftypes.Value) (stringLike, error) {
			called++
			return "stringLike", nil
		}))
		require.NoError(t, err)
		assert.Equal(t, 1, called)
		assert.Equal(t, testStruct{C: "stringLike"}, out)
	})

	t.Run("string-like WithValueFromConverter", func(t *testing.T) {
		type stringLike string
		type stringLikeStruct struct {
			V stringLike `tfsdk:"v"`
		}
		type testStruct struct {
			A []stringLike `tfsdk:"a"`
			B []string     `tfsdk:"b"`
			C stringLike   `tfsdk:"c"`
			D *stringLike  `tfsdk:"d"`
			E string       `tfsdk:"e"`
		}
		stringLikeType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"v": tftypes.String}}
		stringLikeArrayType := tftypes.List{ElementType: stringLikeType}
		stringArrayType := tftypes.List{ElementType: tftypes.String}
		var called int
		out, err := ValueFrom(testStruct{}, WithValueFromConverterFor[stringLike](
			ValueFromConverterForTypedNilHandler[stringLikeStruct](),
			func(attributePath path.Path, in stringLike) (tftypes.Value, error) {
				called++
				return ValueFrom(stringLikeStruct{in})
			},
		))
		require.NoError(t, err)
		assert.Equal(t, 1, called)
		assert.Equal(t, tftypes.NewValue(
			tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"a": stringLikeArrayType, "b": stringArrayType,
				"c": stringLikeType, "d": stringLikeType, "e": tftypes.String,
			}},
			map[string]tftypes.Value{
				"a": tftypes.NewValue(stringLikeArrayType, nil),
				"b": tftypes.NewValue(stringArrayType, nil),
				"c": tftypes.NewValue(stringLikeType, map[string]tftypes.Value{
					"v": tftypes.NewValue(tftypes.String, ""),
				}),
				"d": tftypes.NewValue(stringLikeType, nil),
				"e": tftypes.NewValue(tftypes.String, ""),
			},
		), out)
	})
}

func ptrTo[T any](v T) *T {
	return &v
}
