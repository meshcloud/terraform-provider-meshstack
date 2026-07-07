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
