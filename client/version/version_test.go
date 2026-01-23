package version

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	assertErrorContainsAllOf := func(contains ...string) assert.ErrorAssertionFunc {
		return func(t assert.TestingT, err error, msgAndArgs ...interface{}) bool {
			assert.NotEmpty(t, contains)
			allOk := true
			for _, contain := range contains {
				ok := assert.ErrorContains(t, err, contain, msgAndArgs...)
				allOk = allOk && ok
			}
			return allOk
		}
	}
	tests := []struct {
		name    string
		s       string
		want    Version
		wantErr assert.ErrorAssertionFunc
	}{
		{"valid 1.0.0", "1.0.0", Version{1, 0, 0}, assert.NoError},
		{"valid 1.3.2", "1.3.2", Version{1, 3, 2}, assert.NoError},
		{"not enough parts", "1.1", Version{}, assertErrorContainsAllOf("cannot parse '1.1' as version: expected 3, got 2 fields separated by '.'")},
		{"negative minor", "1.-1.0", Version{}, assertErrorContainsAllOf("cannot parse '1.-1.0' as version: part i=1: negative number '-1' not allowed")},
		{"not a number", "1.1.x", Version{}, assertErrorContainsAllOf(`cannot parse '1.1.x' as version: part i=2: strconv.Atoi: parsing "x": invalid syntax`)},
		{"number too large", "100000000000000000000.1.0", Version{}, assertErrorContainsAllOf(`cannot parse '100000000000000000000.1.0' as version: part i=0: strconv.Atoi: parsing "100000000000000000000": value out of range`)},
		{"multiple errors", "y.x.1", Version{}, assertErrorContainsAllOf(`part i=0: strconv.Atoi: parsing "y": invalid syntax`, `part i=1: strconv.Atoi: parsing "x": invalid syntax`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotV, err := Parse(tt.s)
			if !tt.wantErr(t, err, fmt.Sprintf("Parse(%v)", tt.s)) {
				return
			}
			assert.Equalf(t, tt.want, gotV, "Parse(%v)", tt.s)
		})
	}
}

func TestMustParse(t *testing.T) {
	assert.NotPanics(t, func() {
		MustParse("1.0.0")
	})
	assert.Panics(t, func() {
		MustParse("1.x.0")
	})
}

func TestVersion_Compare(t *testing.T) {
	tests := []struct {
		v, other string
		want     int
	}{
		{"0.0.0", "0.0.0", 0},
		{"0.1.0", "0.1.0", 0},
		{"1.1.0", "0.1.0", 1},
		{"1.1.12312331222", "2.1.0", -1},
		{"1.2.0", "1.3.0", -1},
		{"1.2.1", "1.2.0", 1},
	}
	for _, tt := range tests {
		symbol := "=="
		if tt.want < 0 {
			symbol = "<"
		} else if tt.want > 0 {
			symbol = ">"
		}
		t.Run(fmt.Sprintf("%s %s %s", tt.v, symbol, tt.other), func(t *testing.T) {
			v, err := Parse(tt.v)
			require.NoError(t, err)
			other, err := Parse(tt.other)
			require.NoError(t, err)
			cmp := v.Compare(other)
			assert.Equal(t, tt.want, cmp)
			if cmp < 0 {
				assert.True(t, v.Less(other))
			} else {
				assert.False(t, v.Less(other))
			}
		})
	}
}

func TestVersion_String(t *testing.T) {
	assert.Equal(t, "1.2.3", Version{1, 2, 3}.String())
}
