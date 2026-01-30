package reflect

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type capturingVisitor struct {
	T     *testing.T
	Root  any
	Paths []string
}

func (visitor *capturingVisitor) CaptureAndTraverse(path WalkPath, v reflect.Value) error {
	if !path.IsRoot() {
		visitor.Paths = append(visitor.Paths, path.String())
		traversedValue, err := path.TryTraverse(visitor.Root)
		require.NoError(visitor.T, err)
		assert.Equalf(visitor.T, traversedValue.Interface(), v.Interface(), "traversed value %s not equal to current %s at %s",
			traversedValue, v, path.String())
	}
	return nil
}

func TestWalk(t *testing.T) {
	type (
		//nolint:unused
		testStruct1 struct {
			F1          string
			F2          *int
			ignored     any
			alsoIgnored any
		}
		//nolint:unused
		testStruct2 struct {
			*testStruct1
			F3      []string
			ignored any
		}

		// Nested structures
		innerStruct struct {
			Name  string
			Value *int
		}
		middleStruct struct {
			Inner  *innerStruct
			Inners []innerStruct
		}
		outerStruct struct {
			Middle *middleStruct
			Items  []*innerStruct
		}

		// Structs with maps
		mapStruct struct {
			StringMap map[string]string
			IntMap    map[int]*string
			StructMap map[string]*innerStruct
		}

		// Structs with slices of various types
		sliceStruct struct {
			Strings       []string
			Pointers      []*string
			Structs       []innerStruct
			StructPtrs    []*innerStruct
			InterfaceVals []any
		}

		// Complex combinations
		complexStruct struct {
			MapOfSlices   map[string][]string
			SliceOfMaps   []map[string]int
			MapOfStructs  map[string]innerStruct
			PtrToSlice    *[]string
			PtrToMap      *map[string]int
			NilPtrToSlice *[]string
			NilPtrToMap   *map[string]int
		}
	)

	var (
		v1          = "v1"
		i42         = 42
		str1        = "value1"
		str2        = "value2"
		int1        = 10
		int2        = 20
		stringSlice = []string{"a", "b"}
		intMap      = map[string]int{"x": 1, "y": 2}
	)

	tests := []struct {
		name      string
		v         any
		wantPaths []string
	}{
		{"nil", nil, nil},
		{"top-level value", true, nil},
		{"top-level pointer", &v1, []string{"*"}},
		{"two strings", []string{v1, "v2"}, []string{"[0]", "[1]"}},
		{"slice with pointers", []any{&v1, "v2", &v1}, []string{"[0]", "*[0]", "**[0]", "[1]", "*[1]", "[2]", "*[2]", "**[2]"}},
		{"struct", testStruct1{}, []string{".F1", ".F2", ".*F2"}},
		{"embedded struct nil", testStruct2{}, []string{".F1", ".F2", ".*F2", ".F3", ".F3[<>]"}},
		{"embedded struct non-nil", testStruct2{testStruct1: &testStruct1{}}, []string{".F1", ".F2", ".*F2", ".F3", ".F3[<>]"}},
		{"struct with non-nil pointer", testStruct1{F2: &i42}, []string{".F1", ".F2", ".*F2"}},
		{"struct with values", testStruct1{F1: "hello", F2: &i42}, []string{".F1", ".F2", ".*F2"}},
		{
			name: "nested struct all nil",
			v:    outerStruct{},
			wantPaths: []string{
				".Items", ".Items[<>]",
				".Items*[<>]",
				".Items*[<>].Name",
				".Items*[<>].Value", ".Items*[<>].*Value",
				".Middle", ".*Middle",
				".*Middle.Inner", ".*Middle.*Inner",
				".*Middle.*Inner.Name",
				".*Middle.*Inner.Value", ".*Middle.*Inner.*Value",
				".*Middle.Inners",
				".*Middle.Inners[<>]",
				".*Middle.Inners[<>].Name",
				".*Middle.Inners[<>].Value", ".*Middle.Inners[<>].*Value",
			},
		},
		{
			name: "nested struct with values",
			v: outerStruct{
				Middle: &middleStruct{
					Inner: &innerStruct{Name: "test", Value: &int1},
					Inners: []innerStruct{
						{Name: "first"},
						{Name: "second", Value: &int2},
					},
				},
				Items: []*innerStruct{
					{Name: "item1"},
					nil,
					{Name: "item3", Value: &int1},
				},
			},
			wantPaths: []string{
				".Items",
				".Items[0]", ".Items*[0]", ".Items*[0].Name",
				".Items*[0].Value", ".Items*[0].*Value",
				".Items[1]", ".Items*[1]", ".Items*[1].Name",
				".Items*[1].Value", ".Items*[1].*Value",
				".Items[2]", ".Items*[2]", ".Items*[2].Name",
				".Items*[2].Value", ".Items*[2].*Value",
				".Middle", ".*Middle",
				".*Middle.Inner", ".*Middle.*Inner",
				".*Middle.*Inner.Name",
				".*Middle.*Inner.Value", ".*Middle.*Inner.*Value",
				".*Middle.Inners",
				".*Middle.Inners[0]", ".*Middle.Inners[0].Name",
				".*Middle.Inners[0].Value", ".*Middle.Inners[0].*Value",
				".*Middle.Inners[1]", ".*Middle.Inners[1].Name",
				".*Middle.Inners[1].Value", ".*Middle.Inners[1].*Value",
			},
		},
		{
			name: "map with string keys empty",
			v:    mapStruct{},
			wantPaths: []string{
				".IntMap", ".IntMap[<>]", ".IntMap*[<>]",
				".StringMap", ".StringMap[<>]",
				".StructMap", ".StructMap[<>]",
				".StructMap*[<>]",
				".StructMap*[<>].Name",
				".StructMap*[<>].Value", ".StructMap*[<>].*Value",
			},
		},
		{
			name: "map with string keys populated",
			v: mapStruct{
				StringMap: map[string]string{"key1": "val1", "key2": "val2"},
				IntMap:    map[int]*string{1: &str1, 2: nil},
				StructMap: map[string]*innerStruct{"a": {Name: "struct1"}, "b": nil},
			},
			wantPaths: []string{
				".IntMap",
				".IntMap[1]", ".IntMap*[1]",
				".IntMap[2]", ".IntMap*[2]",
				".StringMap",
				".StringMap[key1]", ".StringMap[key2]",
				".StructMap",
				".StructMap[a]", ".StructMap*[a]", ".StructMap*[a].Name",
				".StructMap*[a].Value", ".StructMap*[a].*Value",
				".StructMap[b]", ".StructMap*[b]", ".StructMap*[b].Name",
				".StructMap*[b].Value", ".StructMap*[b].*Value",
			},
		},
		{
			name: "slices of various types empty",
			v:    sliceStruct{},
			wantPaths: []string{
				".InterfaceVals", ".InterfaceVals[<>]",
				".Pointers", ".Pointers[<>]", ".Pointers*[<>]",
				".Strings", ".Strings[<>]",
				".StructPtrs", ".StructPtrs[<>]",
				".StructPtrs*[<>]",
				".StructPtrs*[<>].Name",
				".StructPtrs*[<>].Value", ".StructPtrs*[<>].*Value",
				".Structs", ".Structs[<>]",
				".Structs[<>].Name",
				".Structs[<>].Value", ".Structs[<>].*Value",
			},
		},
		{
			name: "slices of various types populated",
			v: sliceStruct{
				Strings:    []string{"a", "b"},
				Pointers:   []*string{&str1, nil, &str2},
				Structs:    []innerStruct{{Name: "s1"}, {Name: "s2", Value: &int1}},
				StructPtrs: []*innerStruct{{Name: "p1"}, nil},
				InterfaceVals: []any{
					"string",
					42,
					&str1,
					innerStruct{Name: "interface"},
				},
			},
			wantPaths: []string{
				".InterfaceVals",
				".InterfaceVals[0]", ".InterfaceVals*[0]",
				".InterfaceVals[1]", ".InterfaceVals*[1]",
				".InterfaceVals[2]", ".InterfaceVals*[2]", ".InterfaceVals**[2]",
				".InterfaceVals[3]", ".InterfaceVals*[3]",
				".InterfaceVals*[3].Name",
				".InterfaceVals*[3].Value", ".InterfaceVals*[3].*Value",
				".Pointers",
				".Pointers[0]", ".Pointers*[0]",
				".Pointers[1]", ".Pointers*[1]",
				".Pointers[2]", ".Pointers*[2]",
				".Strings", ".Strings[0]", ".Strings[1]",
				".StructPtrs",
				".StructPtrs[0]", ".StructPtrs*[0]", ".StructPtrs*[0].Name",
				".StructPtrs*[0].Value", ".StructPtrs*[0].*Value",
				".StructPtrs[1]", ".StructPtrs*[1]", ".StructPtrs*[1].Name",
				".StructPtrs*[1].Value", ".StructPtrs*[1].*Value",
				".Structs",
				".Structs[0]", ".Structs[0].Name", ".Structs[0].Value", ".Structs[0].*Value",
				".Structs[1]", ".Structs[1].Name", ".Structs[1].Value", ".Structs[1].*Value",
			},
		},
		{
			name: "complex nested combinations",
			v: complexStruct{
				MapOfSlices: map[string][]string{
					"slice1": {"a", "b"},
					"slice2": {},
				},
				SliceOfMaps: []map[string]int{
					{"x": 1, "y": 2},
					{},
				},
				MapOfStructs: map[string]innerStruct{
					"s1": {Name: "struct1"},
					"s2": {Name: "struct2", Value: &int1},
				},
				PtrToSlice: &stringSlice,
				PtrToMap:   &intMap,
			},
			wantPaths: []string{
				".MapOfSlices",
				".MapOfSlices[slice1]", ".MapOfSlices[slice1][0]", ".MapOfSlices[slice1][1]",
				".MapOfSlices[slice2]", ".MapOfSlices[slice2][<>]",
				".MapOfStructs",
				".MapOfStructs[s1]", ".MapOfStructs[s1].Name",
				".MapOfStructs[s1].Value", ".MapOfStructs[s1].*Value",
				".MapOfStructs[s2]", ".MapOfStructs[s2].Name",
				".MapOfStructs[s2].Value", ".MapOfStructs[s2].*Value",
				".NilPtrToMap", ".*NilPtrToMap", ".*NilPtrToMap[<>]",
				".NilPtrToSlice", ".*NilPtrToSlice", ".*NilPtrToSlice[<>]",
				".PtrToMap", ".*PtrToMap", ".*PtrToMap[x]", ".*PtrToMap[y]",
				".PtrToSlice", ".*PtrToSlice", ".*PtrToSlice[0]", ".*PtrToSlice[1]",
				".SliceOfMaps",
				".SliceOfMaps[0]", ".SliceOfMaps[0][x]", ".SliceOfMaps[0][y]",
				".SliceOfMaps[1]", ".SliceOfMaps[1][<>]",
			},
		},
		{
			name: "nil pointers in various positions",
			v: outerStruct{
				Middle: nil,
				Items:  nil,
			},
			wantPaths: []string{
				".Items", ".Items[<>]",
				".Items*[<>]",
				".Items*[<>].Name",
				".Items*[<>].Value", ".Items*[<>].*Value",
				".Middle", ".*Middle",
				".*Middle.Inner", ".*Middle.*Inner",
				".*Middle.*Inner.Name",
				".*Middle.*Inner.Value", ".*Middle.*Inner.*Value",
				".*Middle.Inners",
				".*Middle.Inners[<>]",
				".*Middle.Inners[<>].Name",
				".*Middle.Inners[<>].Value", ".*Middle.Inners[<>].*Value",
			},
		},
		{
			name: "empty slices and maps",
			v:    sliceStruct{Strings: []string{}, Pointers: []*string{}},
			wantPaths: []string{
				".InterfaceVals", ".InterfaceVals[<>]",
				".Pointers", ".Pointers[<>]", ".Pointers*[<>]",
				".Strings", ".Strings[<>]",
				".StructPtrs", ".StructPtrs[<>]",
				".StructPtrs*[<>]",
				".StructPtrs*[<>].Name",
				".StructPtrs*[<>].Value", ".StructPtrs*[<>].*Value",
				".Structs", ".Structs[<>]",
				".Structs[<>].Name",
				".Structs[<>].Value", ".Structs[<>].*Value",
			},
		},
		{
			name: "deeply nested nil pointers",
			v: &outerStruct{
				Middle: &middleStruct{
					Inner:  nil,
					Inners: nil,
				},
			},
			wantPaths: []string{
				"*",
				"*.Items", "*.Items[<>]",
				"*.Items*[<>]",
				"*.Items*[<>].Name",
				"*.Items*[<>].Value", "*.Items*[<>].*Value",
				"*.Middle", "*.*Middle",
				"*.*Middle.Inner", "*.*Middle.*Inner",
				"*.*Middle.*Inner.Name",
				"*.*Middle.*Inner.Value", "*.*Middle.*Inner.*Value",
				"*.*Middle.Inners", "*.*Middle.Inners[<>]",
				"*.*Middle.Inners[<>].Name",
				"*.*Middle.Inners[<>].Value", "*.*Middle.Inners[<>].*Value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			visitor := capturingVisitor{T: t, Root: tt.v}
			require.NoError(t, Walk(reflect.ValueOf(tt.v), (&visitor).CaptureAndTraverse,
				VisitEmptyContainers(),
				VisitEmbeddedNilStructs(),
			))
			assert.Equal(t, tt.wantPaths, visitor.Paths)
			if tt.v != nil {
				assert.NotNil(t, visitor.Root)
			}
		})
	}
}
