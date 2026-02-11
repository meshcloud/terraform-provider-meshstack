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
	assert.Contains(t, diags[0].Detail(), "key path *[implementation]*[terraform]*[sshPrivateKey][plaintext] matches one of disallowed keys [plaintext")
}
