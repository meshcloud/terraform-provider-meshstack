package hash

import (
	"hash"
	"math"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasher_Hash_unsupported(t *testing.T) {
	t.Run("struct", func(t *testing.T) {
		_, err := Hasher{}.Hash(struct{}{})
		require.ErrorContains(t, err, "cannot handle kind struct")
	})
	t.Run("func", func(t *testing.T) {
		_, err := Hasher{}.Hash(TestHasher_Hash_unsupported)
		require.ErrorContains(t, err, "cannot handle kind func")
	})
	t.Run("chan", func(t *testing.T) {
		_, err := Hasher{}.Hash(make(chan string))
		require.ErrorContains(t, err, "cannot handle kind chan")
	})
}

func TestHasher_Hash(t *testing.T) {
	var nonUniqueHashes []string
	addNonUniqueHash := func(hash string) string {
		nonUniqueHashes = append(nonUniqueHashes, hash)
		return hash
	}
	var (
		emptyMapHash   = addNonUniqueHash("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
		emptyArrayHash = addNonUniqueHash("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")

		setOfValuesHash  = addNonUniqueHash("40be4494cd11c95ca2d719e88a3b1a3d702d33504a7f7b2aa7254ce8b73dda23")
		setOfStringsHash = addNonUniqueHash("01dff5628987c9ae519243ad7bcf8cd7277443458fbeeaab0bb776abaec0aa71")

		nestedMapNilHash = addNonUniqueHash("2df88587ea61710765e5aff00de725281022e1d2cc59c31fe51b7f7776bd5044")
	)

	tests := []struct {
		name  string
		value any
		want  string
	}{
		{"int zero", 0, "b1535c7783ea8829b6b0cf67704539798b4d16c39bf0bfe09494c5d9f12eee30"},
		{"int8 zero", int8(0), "59d5966c96af7ecad5c9d2918d6582d102b2c67f6b765ea28ac24371ab4f93be"},
		{"int 42", 42, "03fa3acc95c4761e105fc5b2c001078919b01cc86d35d6dece677a56fb7e553e"},
		{"int max", int64(math.MaxInt64), "b2ee011c9976b356989192e2b2b768032a663f7850c4f1bb1a74222cfa8827da"},
		{"uint max", uint64(math.MaxUint64), "fbd81d6ae8e531ac82528821e98bf76c1d5ac5ec636aececebe8bd9e691be836"},

		{"float64 zero", 0.0, "22cfca64082af8b7ad33d7263cc675b55531d8d0abd5f32d3554b0c8e25aca9a"},
		{"float64 1.0", 1.0, "ff2cdd13768102caa21b40d17b00ccc0fcd1b953cab2b3d4b27adc805d5e5230"},
		{"float64 1.1", 1.1, "7ee825b5b12c3f4a1a4191709c21b1f672e690ee077ef721002b3497a58a98ce"},
		{"float32 zero", float32(0), "38527ddf79e200d842ade1bf06f25945a6bd154fda9937afea30dc31d28358f0"},
		{"float32 1.1", float32(1.1), "daa67f1bb4cbd709acbbb2369e8a27b177acc54ad5c0e5acc5f945c9156d5e33"},

		{"complex zero", complex(0.0, 0.0), "7a06e4e6da128e3f3b86519181c45d4fea1677eea62c1fadebb281892e650331"},
		{"complex32 zero", complex(float32(0.0), float32(0.0)), "7aedea93543bd2743bc5c17b369c458fade775de8b29af3c842791bec1f23222"},
		{"complex non-zero", complex(1.0, 2.0), "914fa09083b3f7fc132d8dcaa323171072ce54f8c79706031818063adb078d34"},

		{"bool true", true, "f52f3a746c2545658e1c6add32e5410365553ebaaa0433f5f8bd90c6f85fd6e2"},
		{"bool false", false, "a536aa3cede6ea3c1f3e0357c3c60e0f216a8c89b853df13b29daa8f85065dfb"},

		{"empty string", "", "cbb032642036ec7043fa4529f06c9c9d8b12fa70ea6799a19ca8321a808d86fa"},

		{"nil byte slice", []byte(nil), emptyArrayHash},
		{"empty string slice", []string{}, emptyArrayHash},
		{"empty int slice", []int{}, emptyArrayHash},

		{"int slice with zero", []int{0}, "8ebd516f9400ed6c01eb3b6dfad4f7526675f0c25440961bd874b4b3cb8fcd50"},
		{"bool slice with false", []bool{false}, "b392e2ba7c94c0b8dab0faaa8e9a082cdbe62aa77abbb54cf5e19c17190d3eb0"},
		{"string slice with empty string", []string{""}, "453cca57ac707f795e652f00cca0b4156d5fcb5b6a0ce2aa4557d00b4f5d121e"},
		{"string slice b,aa", []string{"b", "aa"}, "61582a5e59c7e22bc578fbd02fc8f212f6b6fcf507ef69a7392edcf8e00c0276"},

		{"string slice a,b", []string{"a", "b"}, setOfStringsHash},
		{"string slice b,a", []string{"b", "a"}, setOfStringsHash},

		{"mixed slice order 1", []any{true, 9, "c"}, setOfValuesHash},
		{"mixed slice order 2", []any{true, "c", 9}, setOfValuesHash},
		{"mixed slice order 3", []any{9, "c", true}, setOfValuesHash},

		{"empty string map", map[string]any{}, emptyMapHash},
		{"nil string map", map[string]any(nil), emptyMapHash},
		{"nil int map", map[int]any(nil), emptyMapHash},
		{"empty string map duplicate", map[string]any{}, emptyMapHash},

		// Maps with one key
		{"map with one key", map[string]any{"key1": "value1"}, "b7c8cd873ef90c786701d97420455991a6ef735ad06700721a934ff3b742ef7a"},
		{"map with nil value", map[string]any{"key": nil}, "debcea1f166010e428df48387304e45c0bf7843a1fa7f1beb38f852228df789b"},

		// Maps with three keys
		{"map with three keys", map[string]any{"key1": "value1", "key2": 42, "key3": true}, "8fc45eb447f153d2e36130cc5fe1e96d5b06b4e8dc45acbf4873571b55323601"},

		// Nested maps
		{"nested map with nil value", map[string]any{"outer": map[string]any{"inner": nil}}, nestedMapNilHash},
		{"nested map with empty value", map[string]any{"outer": map[string]any{"inner": map[string]any{}}}, nestedMapNilHash},
		{"nested map with empty string pointer", map[string]any{"outer": map[string]any{"inner": func() *string { s := ""; return &s }()}}, "68d25e73adcdbdbd079c051e2367fdb31cad9658b11c07a7ff196106c5cb792f"},
		{"nested map with non-empty string pointer", map[string]any{"outer": map[string]any{"inner": func() *string { s := "not-empty"; return &s }()}}, "da604108fc59fb7b34a27eaba98ab6efade2527911efb6798ff8c2fed3ace130"},
	}
	var allHashes []string
	allOk := true
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := Hasher{}.Hash(tt.value)
			require.NoError(t, err)
			actualHex := actual.Hex()
			require.NotEmpty(t, actualHex)
			allHashes = append(allHashes, actualHex)
			ok := assert.Equal(t, tt.want, actualHex)
			allOk = allOk && ok
		})
	}
	if !allOk {
		return
	}
	t.Run("nonUniqueHashes are correctly tracked", func(t *testing.T) {
		require.NotEmpty(t, nonUniqueHashes)
		require.NotEmpty(t, allHashes)

		allHashes = slices.DeleteFunc(allHashes, func(s string) bool {
			return slices.Contains(nonUniqueHashes, s)
		})
		slices.Sort(allHashes)
		assert.Equal(t, allHashes, slices.Compact(slices.Clone(allHashes)))
	})
}

func TestHasher_Hash_writtenBytes(t *testing.T) {
	// Helper to create expected byte sequences
	mkBytes := func(bytes ...byte) *writtenBytesHash {
		h := writtenBytesHash(bytes)
		return &h
	}

	tests := []struct {
		name  string
		value any
		want  []*writtenBytesHash
	}{
		{
			"string slice b,a",
			[]string{"b", "a"},
			[]*writtenBytesHash{
				mkBytes(0x17, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03),
				mkBytes(0x18, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x62),
				mkBytes(0x18, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x61),
			},
		},
		{
			"string slice b,aa",
			[]string{"b", "aa"},
			[]*writtenBytesHash{
				mkBytes(0x17, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x18, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x62, 0x61),
				mkBytes(0x18, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x62),
				mkBytes(0x18, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x61, 0x61),
			},
		},
		{
			"string slice with empty string",
			[]string{""},
			[]*writtenBytesHash{
				mkBytes(0x17, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x18, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00),
				mkBytes(0x18, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00),
			},
		},
		{
			"int slice with zero",
			[]int{0},
			[]*writtenBytesHash{
				mkBytes(0x17, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00),
				mkBytes(0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00),
			},
		},

		{
			"int max",
			int64(math.MaxInt64),
			[]*writtenBytesHash{
				mkBytes(0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f),
			},
		},
		{
			"uint max",
			uint64(math.MaxUint64),
			[]*writtenBytesHash{
				mkBytes(0x0b, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff),
			},
		},
		{
			"float32 1.1",
			float32(1.1),
			[]*writtenBytesHash{
				mkBytes(0x0d, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xa0, 0x99, 0x99, 0xf1, 0x3f),
			},
		},
		{
			"float64 1.1",
			1.1,
			[]*writtenBytesHash{
				mkBytes(0x0e, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x9a, 0x99, 0x99, 0x99, 0x99, 0x99, 0xf1, 0x3f),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var writtenBytes []*writtenBytesHash
			_, err := Hasher{Underlying: func() hash.Hash {
				h := &writtenBytesHash{}
				writtenBytes = append(writtenBytes, h)
				return h
			}}.Hash(tt.value)
			require.NoError(t, err)
			assert.Equal(t, tt.want, writtenBytes)
		})
	}
}

type writtenBytesHash []byte

func (c *writtenBytesHash) Write(p []byte) (n int, err error) {
	*c = append(*c, p...)
	return len(p), nil
}

func (c *writtenBytesHash) Sum(b []byte) []byte {
	return append(b, *c...)
}

func (c *writtenBytesHash) Reset() {
	panic("not implemented for test")
}

func (c *writtenBytesHash) Size() int {
	panic("not implemented for test")
}

func (c *writtenBytesHash) BlockSize() int {
	panic("not implemented for test")
}

func TestHasher_Hash_disallowedMapKeys(t *testing.T) {
	v := []any{
		1.0,
		&map[string]any{"key1": "value1", "key2": map[string]any{"superillegal": true}},
	}
	_, err := Hasher{}.Hash(v, DisallowMapKeys("superillegal"))
	require.ErrorContains(t, err, "key path **[1]*[key2][superillegal] matches one of disallowed keys [superillegal]")
}
