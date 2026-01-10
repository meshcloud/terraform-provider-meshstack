package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type MeshBuildingBlock struct{}
type MeshBuildingBlockV2 struct{}
type MeshProject struct{}
type MeshWorkspace struct{}
type MeshProjectBinding struct{}
type MeshProjectGroupBinding struct {
	MeshProjectBinding
}
type MeshProjectUserBinding struct {
	MeshProjectBinding
}
type MeshWorkspaceGroupBinding struct{}
type MeshWorkspaceUserBinding struct{}

func TestInferMeshObjectName(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func() string
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
			expected: "meshBuildingBlockV2",
		},
		{
			name:     "MeshProject",
			testFunc: inferMeshObjectName[MeshProject],
			expected: "meshProject",
		},
		{
			name:     "MeshWorkspace",
			testFunc: inferMeshObjectName[MeshWorkspace],
			expected: "meshWorkspace",
		},
		{
			name:     "MeshProjectBinding",
			testFunc: inferMeshObjectName[MeshProjectBinding],
			expected: "meshProjectBinding",
		},
		{
			name:     "MeshProjectGroupBinding (embedded struct)",
			testFunc: inferMeshObjectName[MeshProjectGroupBinding],
			expected: "meshProjectGroupBinding",
		},
		{
			name:     "MeshProjectUserBinding (embedded struct)",
			testFunc: inferMeshObjectName[MeshProjectUserBinding],
			expected: "meshProjectUserBinding",
		},
		{
			name:     "MeshWorkspaceGroupBinding (embedded struct)",
			testFunc: inferMeshObjectName[MeshWorkspaceGroupBinding],
			expected: "meshWorkspaceGroupBinding",
		},
		{
			name:     "MeshWorkspaceUserBinding (embedded struct)",
			testFunc: inferMeshObjectName[MeshWorkspaceUserBinding],
			expected: "meshWorkspaceUserBinding",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.testFunc()
			assert.Equal(t, tt.expected, actual)
		})
	}
}
