package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type MeshBuildingBlock struct{}
type MeshBuildingBlockV2 struct{}
type MeshTenantV4 struct{}
type MeshWorkspace struct{}

func TestInferMeshObjectName(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func() (string, string)
		expected string
	}{
		{
			name:     "MeshBuildingBlock",
			testFunc: inferMeshObjectName[MeshBuildingBlock],
			expected: "meshBuildingBlock",
		},
		{
			name:     "MeshBuildingBlockV2",
			testFunc: inferMeshObjectName[MeshBuildingBlockV2],
			expected: "meshBuildingBlock",
		},
		{
			name:     "MeshWorkspace",
			testFunc: inferMeshObjectName[MeshWorkspace],
			expected: "meshWorkspace",
		},
		{
			name:     "MeshTenantV4",
			testFunc: inferMeshObjectName[MeshTenantV4],
			expected: "meshTenant",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, _ := tt.testFunc()
			assert.Equal(t, tt.expected, actual)
		})
	}
}
