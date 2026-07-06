package provider

import (
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

var (
	//go:embed testdata/bbd/version-spec.json
	versionSpecJson []byte
	//go:embed testdata/bbd/version-spec-irrelevant-change.json
	versionSpecIrrelevantChangeJson []byte
	//go:embed testdata/bbd/version-spec-relevant-change.json
	versionSpecRelevantChangeJson []byte
	//go:embed testdata/bbd/version-spec-with-plaintext-secret.json
	versionSpecPlaintextSecretJson []byte
	//go:embed testdata/bbd/version-spec-null-outputs.json
	versionSpecNullOutputsChangeJson []byte
	//go:embed testdata/bbd/version-spec-empty-outputs.json
	versionSpecEmptyOutputsJson []byte
)

func Test_versionContentHash(t *testing.T) {
	// If constant values below are required to change, you need a good reason and consider backwards compatibility!
	const (
		hashWhichShouldNeverChange1 = "djI6Njc0Yzc3YzI4ZTRlYjRjZWM5OWI5ZjFlNzNhZDExYjUyMGEzNjdkYTQxNmZmM2ZhOTBkNWU1NDI2ZTA5YmVmYw=="
		hashWhichShouldNeverChange2 = "djI6MDIwY2Q3YzAzMmZmNmNlMDVhMDI4NzNmZDMxOWU3YjcyMDY4OTZmYTQxNTkwNDc5MTgzNmM5MzNmMGEyMzllZQ=="
		hashWhichShouldNeverChange3 = "djI6MmI5OTJlMjM0MzE2YmFhMDhkMWY0ZDAxN2Y1NzM3OWRkZmI0NWYzOWYzYWU3YjE0ODA3OTMyMjkzOGRmZDhlMA=="
	)
	require.NotEqual(t, hashWhichShouldNeverChange1, hashWhichShouldNeverChange2)
	tests := []struct {
		name string
		json []byte
		want string
	}{
		{"example", versionSpecJson, hashWhichShouldNeverChange1},
		{"example with irrelevant changes", versionSpecIrrelevantChangeJson, hashWhichShouldNeverChange1},
		{"example with relevant changes", versionSpecRelevantChangeJson, hashWhichShouldNeverChange2},
		{"example with null outputs", versionSpecNullOutputsChangeJson, hashWhichShouldNeverChange3},
		{"example with empty outputs (same hash)", versionSpecEmptyOutputsJson, hashWhichShouldNeverChange3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var versionSpec client.MeshBuildingBlockDefinitionVersionSpec
			require.NoError(t, json.Unmarshal(tt.json, &versionSpec))
			var diags diag.Diagnostics
			actualHashStr := calculateBuildingBlockDefinitionVersionContentHash(versionSpec, &diags).toBase64()
			require.Empty(t, diags)
			require.Equal(t, tt.want, actualHashStr)
		})
	}
}

func Test_versionContentHash_plaintextSecret(t *testing.T) {
	var versionSpec client.MeshBuildingBlockDefinitionVersionSpec
	require.NoError(t, json.Unmarshal(versionSpecPlaintextSecretJson, &versionSpec))
	var diags diag.Diagnostics
	calculateBuildingBlockDefinitionVersionContentHash(versionSpec, &diags)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Detail(), "version_spec carries a plaintext secret value, which must not be hashed")
}

// Test_versionContentHash_userDataNamedPlaintext guards against the false positive from issue #196: an
// input named "plaintext" is a user-chosen map key, not a secret, so it must hash successfully rather than
// trip the plaintext safeguard (the old JSON-key-based safeguard rejected it).
func Test_versionContentHash_userDataNamedPlaintext(t *testing.T) {
	const userDataJson = `{
		"inputs": {
			"plaintext": {
				"displayName": "Plaintext",
				"type": "STRING",
				"assignmentType": "STATIC",
				"argument": "hello"
			}
		},
		"implementation": {"terraform": {}}
	}`
	var versionSpec client.MeshBuildingBlockDefinitionVersionSpec
	require.NoError(t, json.Unmarshal([]byte(userDataJson), &versionSpec))
	var diags diag.Diagnostics
	actualHashStr := calculateBuildingBlockDefinitionVersionContentHash(versionSpec, &diags).toBase64()
	require.Empty(t, diags)
	assert.NotEmpty(t, actualHashStr)
}

// Test_versionContentHash_ignoresPerVersionFields pins the invariant that the content hash is independent
// of the per-BBD buildingBlockDefinitionRef and the per-version versionNumber/state (all stripped before
// hashing). Were they hashed, the same version_spec would hash differently across BBDs/versions, breaking
// the released-version immutability and change-detection comparisons. This is the focused, behavioral
// replacement for the old DisallowMapKeys("buildingBlockDefinitionRef") safeguard, which - like the
// "plaintext" one - would have misfired on a user-chosen input/output named "buildingBlockDefinitionRef".
func Test_versionContentHash_ignoresPerVersionFields(t *testing.T) {
	var base, mutated client.MeshBuildingBlockDefinitionVersionSpec
	require.NoError(t, json.Unmarshal(versionSpecJson, &base))
	require.NoError(t, json.Unmarshal(versionSpecJson, &mutated))

	mutated.BuildingBlockDefinitionRef = &client.BuildingBlockDefinitionRef{
		Kind: client.MeshObjectKind.BuildingBlockDefinition,
		Uuid: "a-different-bbd-uuid",
	}
	mutated.VersionNumber = new(int64(99))
	mutated.State = client.MeshBuildingBlockDefinitionVersionStateReleased.Ptr()

	var diags diag.Diagnostics
	baseHash := calculateBuildingBlockDefinitionVersionContentHash(base, &diags).toBase64()
	mutatedHash := calculateBuildingBlockDefinitionVersionContentHash(mutated, &diags).toBase64()
	require.Empty(t, diags)
	assert.Equal(t, baseHash, mutatedHash, "buildingBlockDefinitionRef/versionNumber/state must not affect the content hash")
}
