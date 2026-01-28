package provider

import (
	_ "embed"
	"encoding/json"
	"testing"

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
)

func Test_versionContentHash(t *testing.T) {
	// If constant values below are required to change, you need a good reason and consider backwards compatibility!
	const (
		hashWhichShouldNeverChange1 = "v1:371852de7e9cb74429a715bf5813cfde788f6ae2dbbc4c5f97251608e47d8d66"
		hashWhichShouldNeverChange2 = "v1:0fc85de94bea5693c6ee69248cdc410677467716bec2bfec09820463918b15ef"
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var versionSpec client.MeshBuildingBlockDefinitionVersionSpec
			require.NoError(t, json.Unmarshal(tt.json, &versionSpec))
			actualHash, err := versionContentHash(versionSpec)
			require.NoError(t, err)
			require.Equal(t, tt.want, actualHash)
		})
	}
}

func Test_versionContentHash_plaintextSecret(t *testing.T) {
	var versionSpec client.MeshBuildingBlockDefinitionVersionSpec
	require.NoError(t, json.Unmarshal(versionSpecPlaintextSecretJson, &versionSpec))
	_, err := versionContentHash(versionSpec)
	require.ErrorContains(t, err, "key '.*implementation.*terraform.*sshPrivateKey.plaintext' must not be in disallowed keys [plaintext]")

}
