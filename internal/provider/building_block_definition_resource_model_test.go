package provider

import (
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

var (
	//go:embed testdata/bbd/version-spec.json
	versionSpecJson []byte
	//go:embed testdata/bbd/version-spec-irrelevant-change.json
	versionSpecIrrelevantChangeJson []byte
	//go:embed testdata/bbd/version-spec-with-reordered-inputs.json
	versionSpecReorderedInputsJson []byte
	//go:embed testdata/bbd/version-spec-with-displayOrder.json
	versionSpecWithDisplayOrderJson []byte
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
		hashWhichShouldNeverChange1 = "djI6N2Y4NDNjY2I0YTUzYjY3OWUyNDVhYzkyODFiM2UyZTk1N2JlNjc0YWNjMGY0OGVmMWM3YjhjMmJhNTJmMzlhOA=="
		hashWhichShouldNeverChange2 = "djI6ODFlNGU0MzZmMmUzNzMwZDBhNWUwZjJlYzY4NTcwZWM4OGNiN2Y4MmI5MzM1MGQ0ZTRkOTVkZTEyMDhkYTU3YQ=="
		hashWhichShouldNeverChange3 = "djI6MWEyNGY3MGRlNjE3ZTY0ZmMyNThjZWNhNGE0YjcxNTllY2ViZDlhOTVhY2Q0ZWE0MDg1MWU0NDNjMDgwMDM4ZQ=="
		hashWhichShouldNeverChange4 = "djI6M2E5ZmUwMTBiZmVmM2RmYmM1MzE3NDJhYjEzMGJjMmM1NTliODZkOWJhNTk1ZDk1MWRjNWYyZmI3YmY2NjI0Yw=="
	)
	require.NotEqual(t, hashWhichShouldNeverChange1, hashWhichShouldNeverChange2)
	tests := []struct {
		name string
		json []byte
		want string
	}{
		{"example", versionSpecJson, hashWhichShouldNeverChange1},
		{"example with reordered inputs", versionSpecReorderedInputsJson, hashWhichShouldNeverChange1},
		{"example with irrelevant changes", versionSpecIrrelevantChangeJson, hashWhichShouldNeverChange1},
		{"example with relevant changes", versionSpecRelevantChangeJson, hashWhichShouldNeverChange2},
		{"example with null outputs", versionSpecNullOutputsChangeJson, hashWhichShouldNeverChange3},
		{"example with empty outputs (same hash)", versionSpecEmptyOutputsJson, hashWhichShouldNeverChange3},
		{"example with display order set (different hash)", versionSpecWithDisplayOrderJson, hashWhichShouldNeverChange4},
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
