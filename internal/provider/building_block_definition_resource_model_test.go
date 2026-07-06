package provider

import (
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
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
		hashWhichShouldNeverChange1 = "v1:674c77c28e4eb4cec99b9f1e73ad11b520a367da416ff3fa90d5e5426e09befc"
		hashWhichShouldNeverChange2 = "v1:020cd7c032ff6ce05a02873fd319e7b7206896fa415904791836c933f0a239ee"
		hashWhichShouldNeverChange3 = "v1:2b992e234316baa08d1f4d017f57379ddfb45f39f3ae7b148079322938dfd8e0"
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
			actualHash := versionContentHash(versionSpec, &diags)
			require.Empty(t, diags)
			require.Equal(t, tt.want, actualHash)
		})
	}
}

func Test_versionContentHash_plaintextSecret(t *testing.T) {
	var versionSpec client.MeshBuildingBlockDefinitionVersionSpec
	require.NoError(t, json.Unmarshal(versionSpecPlaintextSecretJson, &versionSpec))
	var diags diag.Diagnostics
	versionContentHash(versionSpec, &diags)
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
	actualHash := versionContentHash(versionSpec, &diags)
	require.Empty(t, diags)
	assert.NotEmpty(t, actualHash)
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
	baseHash := versionContentHash(base, &diags)
	mutatedHash := versionContentHash(mutated, &diags)
	require.Empty(t, diags)
	assert.Equal(t, baseHash, mutatedHash, "buildingBlockDefinitionRef/versionNumber/state must not affect the content hash")
}

// The backend collapses a null inputs map and an empty one, so SetFromVersionClientDtos must preserve the
// caller's exact shape when the version carries no inputs — otherwise Terraform reports a null-vs-empty
// "Provider produced inconsistent result after apply".
func Test_SetFromVersionClientDtos_preservesInputsShape(t *testing.T) {
	draftState := client.MeshBuildingBlockDefinitionVersionStateDraft.Unwrap()
	versionWithNoInputs := client.MeshBuildingBlockDefinitionVersion{
		Metadata: client.MeshBuildingBlockDefinitionVersionMetadata{Uuid: "v1"},
		Spec: client.MeshBuildingBlockDefinitionVersionSpec{
			VersionNumber: new(int64(1)),
			State:         &draftState,
			Implementation: client.MeshBuildingBlockDefinitionImplementation{
				Terraform: &client.MeshBuildingBlockDefinitionTerraformImplementation{},
			},
			// Inputs left nil, mimicking a backend response that omits an empty inputs map.
		},
	}

	tests := []struct {
		name    string
		prior   map[string]*client.MeshBuildingBlockDefinitionInput
		wantNil bool
	}{
		{"nil inputs stay nil", nil, true},
		{"empty inputs stay empty (not null)", map[string]*client.MeshBuildingBlockDefinitionInput{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &buildingBlockDefinition{}
			model.VersionSpec.Inputs = tt.prior

			var diags diag.Diagnostics
			model.SetFromVersionClientDtos(&diags, generic.KnownValue(true), "bbd", versionWithNoInputs)
			require.False(t, diags.HasError(), "unexpected diags: %v", diags.Errors())

			if tt.wantNil {
				require.Nil(t, model.VersionSpec.Inputs)
			} else {
				require.NotNil(t, model.VersionSpec.Inputs)
				require.Empty(t, model.VersionSpec.Inputs)
			}
		})
	}
}
