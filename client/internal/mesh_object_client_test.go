package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type MeshBuildingBlock struct{}
type MeshBuildingBlockV2 struct{}
type MeshTenantV4 struct{}
type MeshWorkspace struct{}

func Test_inferMeshObjectKindFromType(t *testing.T) {
	tests := []struct {
		kind     string
		testFunc func() (string, string)
		expected string
	}{
		{
			kind:     "MeshBuildingBlock",
			testFunc: inferMeshObjectKindFromType[MeshBuildingBlock],
			expected: "meshBuildingBlock",
		},
		{
			kind:     "MeshBuildingBlockV2",
			testFunc: inferMeshObjectKindFromType[MeshBuildingBlockV2],
			expected: "meshBuildingBlock",
		},
		{
			kind:     "MeshWorkspace",
			testFunc: inferMeshObjectKindFromType[MeshWorkspace],
			expected: "meshWorkspace",
		},
		{
			kind:     "MeshTenantV4",
			testFunc: inferMeshObjectKindFromType[MeshTenantV4],
			expected: "meshTenant",
		},
	}

	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			actual, _ := tt.testFunc()
			assert.Equal(t, tt.expected, actual)
		})
	}
}
