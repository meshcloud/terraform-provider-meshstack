package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMeshTenant_DeletionSuccessful(t *testing.T) {
	tests := []struct {
		name     string
		tenant   *MeshTenant
		wantDone bool
	}{
		{
			name:     "nil (404 — tenant purged)",
			tenant:   nil,
			wantDone: true,
		},
		{
			name: "lifecycle DELETED (deletion completed, tenant still returned)",
			tenant: &MeshTenant{Status: MeshTenantStatus{
				Lifecycle: MeshTenantLifecycle{State: TenantLifecycleStateDeleted},
			}},
			wantDone: true,
		},
		{
			name: "lifecycle MARKED_FOR_DELETION (deletion still running)",
			tenant: &MeshTenant{Status: MeshTenantStatus{
				Lifecycle: MeshTenantLifecycle{
					State:             TenantLifecycleStateMarkedForDeletion,
					MarkedForDeletion: &MeshTenantLifecycleAction{Timestamp: "2026-07-30T16:14:14Z"},
				},
			}},
			wantDone: false,
		},
		{
			name: "lifecycle ACTIVE",
			tenant: &MeshTenant{Status: MeshTenantStatus{
				Lifecycle: MeshTenantLifecycle{State: TenantLifecycleStateActive},
			}},
			wantDone: false,
		},
		{
			name:     "no lifecycle reported",
			tenant:   &MeshTenant{Metadata: MeshTenantMetadata{Uuid: "test-uuid"}},
			wantDone: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			done, err := tt.tenant.DeletionSuccessful()
			assert.Equal(t, tt.wantDone, done)
			assert.NoError(t, err)
		})
	}
}

func TestTenantDeletionState(t *testing.T) {
	assert.Equal(t, tenantNotObserved, (*MeshTenant)(nil).DeletionState())

	assert.Equal(t, "DELETED",
		(&MeshTenant{Status: MeshTenantStatus{
			Lifecycle: MeshTenantLifecycle{State: TenantLifecycleStateDeleted},
		}}).DeletionState(),
	)
	assert.Contains(t,
		(&MeshTenant{Status: MeshTenantStatus{Lifecycle: MeshTenantLifecycle{
			State:             TenantLifecycleStateMarkedForDeletion,
			MarkedForDeletion: &MeshTenantLifecycleAction{Timestamp: "2026-07-30T16:14:14Z"},
		}}}).DeletionState(),
		"MARKED_FOR_DELETION since 2026-07-30T16:14:14Z",
	)
	assert.Contains(t,
		(&MeshTenant{Status: MeshTenantStatus{
			Lifecycle: MeshTenantLifecycle{State: TenantLifecycleStateMarkedForDeletion},
		}}).DeletionState(),
		"MARKED_FOR_DELETION, awaiting",
	)
	assert.Contains(t,
		(&MeshTenant{Status: MeshTenantStatus{
			Lifecycle: MeshTenantLifecycle{State: TenantLifecycleStateActive},
		}}).DeletionState(),
		"has not acted on it",
	)
}
