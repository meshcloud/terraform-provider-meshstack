package types

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/client/types/ptr"
)

func TestSecretOrAny(t *testing.T) {
	type testCase struct {
		name string
		json string
		v    SecretOrAny

		wantX, wantY bool
	}
	tests := []testCase{
		{"empty", `null`, SecretOrAny{}, false, false},
		{"X plaintext", `{"plaintext":"some-secret"}`, SecretOrAny{X: Secret{Plaintext: ptr.To("some-secret")}}, true, false},
		{"Y string", `"some-string"`, SecretOrAny{Y: "some-string"}, false, true},
		{"Y bool", `true`, SecretOrAny{Y: true}, false, true},
		{"Y number", `1.23123`, SecretOrAny{Y: 1.23123}, false, true},
		{"Y empty string", `""`, SecretOrAny{Y: ""}, false, true},
		{"Y other struct", `{"A":"aa","B":"bb"}`, SecretOrAny{Y: map[string]any{"A": "aa", "B": "bb"}}, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("unmarshal", func(t *testing.T) {
				var unmarshalled SecretOrAny
				require.NoError(t, json.Unmarshal([]byte(tt.json), &unmarshalled))
				assert.Equal(t, tt.v, unmarshalled)
				assert.Equal(t, tt.wantX, unmarshalled.HasX())
				assert.Equal(t, tt.wantY, unmarshalled.HasY())
			})

			t.Run("marshal", func(t *testing.T) {
				marshalled, err := json.Marshal(tt.v)
				require.NoError(t, err)
				assert.Equal(t, tt.json, string(marshalled))
			})
		})
	}
}

func TestIsSet(t *testing.T) {
	type (
		someStruct struct {
			A string
		}
		someString string
		someSet    Set[someString]
	)
	tests := []struct {
		name string
		t    reflect.Type
		want bool
	}{
		{"bool", reflect.TypeFor[bool](), false},
		{"any", reflect.TypeFor[any](), false},
		{"int", reflect.TypeFor[any](), false},
		{"some set (not supported)", reflect.TypeFor[someSet](), false},
		{"set of string", reflect.TypeFor[Set[string]](), true},
		{"set of int", reflect.TypeFor[Set[string]](), true},
		{"set of struct", reflect.TypeFor[Set[someStruct]](), true},
		{"set of some string", reflect.TypeFor[Set[someString]](), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, IsSet(tt.t), "IsSet(%v)", tt.t)
		})
	}
}
