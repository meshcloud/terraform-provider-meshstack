package types

import (
	"encoding/json"
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
