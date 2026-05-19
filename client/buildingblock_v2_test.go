package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMeshBuildingBlockV2_DeletionSuccessful(t *testing.T) {
	tests := []struct {
		name     string
		bb       *MeshBuildingBlockV2
		wantDone bool
		wantErr  bool
	}{
		{
			name:     "nil (hard deletion / 404)",
			bb:       nil,
			wantDone: true,
			wantErr:  false,
		},
		{
			name: "lifecycle state DELETED",
			bb: &MeshBuildingBlockV2{
				Status: MeshBuildingBlockV2Status{
					Lifecycle: MeshBuildingBlockV2Lifecycle{State: BUILDING_BLOCK_LIFECYCLE_STATE_DELETED},
				},
			},
			wantDone: true,
			wantErr:  false,
		},
		{
			name: "status FAILED during deletion",
			bb: &MeshBuildingBlockV2{
				Metadata: MeshBuildingBlockV2Metadata{Uuid: "test-uuid"},
				Status: MeshBuildingBlockV2Status{
					Status: BUILDING_BLOCK_STATUS_FAILED,
				},
			},
			wantDone: false,
			wantErr:  true,
		},
		{
			name: "still in progress (MARKED_FOR_DELETION lifecycle, non-failed status)",
			bb: &MeshBuildingBlockV2{
				Status: MeshBuildingBlockV2Status{
					Status:    BUILDING_BLOCK_STATUS_IN_PROGRESS,
					Lifecycle: MeshBuildingBlockV2Lifecycle{State: BUILDING_BLOCK_LIFECYCLE_STATE_MARKED_FOR_DELETION},
				},
			},
			wantDone: false,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			done, err := tt.bb.DeletionSuccessful()
			assert.Equal(t, tt.wantDone, done)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
