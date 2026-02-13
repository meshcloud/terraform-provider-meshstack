package client

import (
	"embed"
	"encoding/json"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/client/types"
	"github.com/meshcloud/terraform-provider-meshstack/client/types/ptr"
)

var (
	//go:embed testdata/building_block_definition_version_input
	bbdInputTestdata embed.FS
)

func TestMeshBuildingBlockDefinitionInput_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name             string
		wantSensitive    bool
		wantArgument     types.SecretOrAny
		wantDefaultValue types.SecretOrAny
		wantErr          assert.ErrorAssertionFunc
	}{
		{"empty", false, types.SecretOrAny{}, types.SecretOrAny{}, assert.NoError},
		{"not_sensitive", false, types.SecretOrAny{Y: true}, types.SecretOrAny{Y: "some-string"}, assert.NoError},
		{"not_sensitive_but_hash", false, types.SecretOrAny{Y: map[string]any{"hash": "some-hash-looks-like-secret"}}, types.SecretOrAny{}, assert.NoError},
		{"sensitive", true, types.SecretOrAny{}, types.SecretOrAny{X: types.Secret{Hash: ptr.To("some-hash")}}, assert.NoError},
		{"sensitive_but_no_hash", true, types.SecretOrAny{Y: map[string]any{}}, types.SecretOrAny{}, func(t assert.TestingT, err error, msgAndArgs ...any) bool {
			return assert.ErrorContains(t, err, "got sensitive argument or default_value but variant Y is set instead")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonFile, err := bbdInputTestdata.ReadFile(path.Join("testdata/building_block_definition_version_input", path.Base(tt.name)+".json"))
			require.NoError(t, err)
			var target MeshBuildingBlockDefinitionInput
			if tt.wantErr(t, json.Unmarshal(jsonFile, &target)) {
				expected := MeshBuildingBlockDefinitionInput{
					IsSensitive:  tt.wantSensitive,
					Argument:     tt.wantArgument,
					DefaultValue: tt.wantDefaultValue,
				}
				assert.Equal(t, expected, target)
			}
		})
	}
}
