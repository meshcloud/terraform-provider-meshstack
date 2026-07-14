package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/client/types/enum"
)

func TestMeshBuildingBlockV2_DeletionSuccessful(t *testing.T) {
	tests := []struct {
		name     string
		bb       *MeshBuildingBlockV2
		wantDone bool
		wantErr  bool
	}{
		{
			name:     "nil (404 — hard deletion / purge)",
			bb:       nil,
			wantDone: true,
			wantErr:  false,
		},
		{
			name: "lifecycle state DELETED (soft delete completed, block still returned)",
			bb: &MeshBuildingBlockV2{
				Status: &MeshBuildingBlockV2Status{
					Lifecycle: MeshBuildingBlockV2Lifecycle{State: BuildingBlockLifecycleStateDeleted},
				},
			},
			wantDone: true,
			wantErr:  false,
		},
		{
			name: "status FAILED during deletion",
			bb: &MeshBuildingBlockV2{
				Metadata: MeshBuildingBlockV2Metadata{Uuid: new("test-uuid")},
				Status: &MeshBuildingBlockV2Status{
					Status: BuildingBlockStatusFailed,
				},
			},
			wantDone: false,
			wantErr:  true,
		},
		{
			name: "status FAILED but force-purged keeps polling (transient, will reach DELETED)",
			bb: &MeshBuildingBlockV2{
				Metadata: MeshBuildingBlockV2Metadata{Uuid: new("test-uuid")},
				Status: &MeshBuildingBlockV2Status{
					Status:     BuildingBlockStatusFailed,
					ForcePurge: true,
				},
			},
			wantDone: false,
			wantErr:  false,
		},
		{
			name: "status FAILED with nil Uuid does not panic",
			bb: &MeshBuildingBlockV2{
				Metadata: MeshBuildingBlockV2Metadata{Uuid: nil},
				Status: &MeshBuildingBlockV2Status{
					Status: BuildingBlockStatusFailed,
				},
			},
			wantDone: false,
			wantErr:  true,
		},
		{
			name: "still in progress (MARKED_FOR_DELETION lifecycle, non-failed status)",
			bb: &MeshBuildingBlockV2{
				Status: &MeshBuildingBlockV2Status{
					Lifecycle: MeshBuildingBlockV2Lifecycle{State: BuildingBlockLifecycleStateMarkedForDeletion},
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

func TestMeshBuildingBlockV2_CreateSuccessful(t *testing.T) {
	tests := []struct {
		name        string
		bb          *MeshBuildingBlockV2
		wantDone    bool
		wantErr     bool
		errContains string
	}{
		{
			name:     "nil (not found after creation)",
			bb:       nil,
			wantDone: false,
			wantErr:  true,
		},
		{
			name:     "no status yet — keep polling",
			bb:       &MeshBuildingBlockV2{Metadata: MeshBuildingBlockV2Metadata{Uuid: new("test-uuid")}},
			wantDone: false,
			wantErr:  false,
		},
		{
			name: "SUCCEEDED",
			bb: &MeshBuildingBlockV2{
				Metadata: MeshBuildingBlockV2Metadata{Uuid: new("test-uuid")},
				Status:   &MeshBuildingBlockV2Status{Status: BuildingBlockStatusSucceeded},
			},
			wantDone: true,
			wantErr:  false,
		},
		{
			name: "FAILED",
			bb: &MeshBuildingBlockV2{
				Metadata: MeshBuildingBlockV2Metadata{Uuid: new("test-uuid")},
				Status:   &MeshBuildingBlockV2Status{Status: BuildingBlockStatusFailed},
			},
			wantDone: false,
			wantErr:  true,
		},
		{
			name: "ABORTED",
			bb: &MeshBuildingBlockV2{
				Metadata: MeshBuildingBlockV2Metadata{Uuid: new("test-uuid")},
				Status:   &MeshBuildingBlockV2Status{Status: BuildingBlockStatusAborted},
			},
			wantDone: false,
			wantErr:  true,
		},
		{
			name: "WAITING_FOR_USER_INPUT — terminal but non-fatal",
			bb: &MeshBuildingBlockV2{
				Metadata: MeshBuildingBlockV2Metadata{Uuid: new("test-uuid")},
				Status:   &MeshBuildingBlockV2Status{Status: BuildingBlockStatusWaitingForUserInput},
			},
			wantDone: true,
			wantErr:  false,
		},
		{
			name: "WAITING_FOR_APPROVAL — terminal but non-fatal",
			bb: &MeshBuildingBlockV2{
				Metadata: MeshBuildingBlockV2Metadata{Uuid: new("test-uuid")},
				Status:   &MeshBuildingBlockV2Status{Status: BuildingBlockStatusWaitingForApproval},
			},
			wantDone: true,
			wantErr:  false,
		},
		{
			name: "FAILED with nil Uuid does not panic",
			bb: &MeshBuildingBlockV2{
				Metadata: MeshBuildingBlockV2Metadata{Uuid: nil},
				Status:   &MeshBuildingBlockV2Status{Status: BuildingBlockStatusFailed},
			},
			wantDone:    false,
			wantErr:     true,
			errContains: "<unknown>",
		},
		{
			name: "unknown status — fail fast",
			bb: &MeshBuildingBlockV2{
				Metadata: MeshBuildingBlockV2Metadata{Uuid: new("test-uuid")},
				Status:   &MeshBuildingBlockV2Status{Status: enum.Entry[BuildingBlockStatus]("SOMETHING_NEW")},
			},
			wantDone:    false,
			wantErr:     true,
			errContains: "unknown building block status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			done, err := tt.bb.CreateSuccessful()
			assert.Equal(t, tt.wantDone, done)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
