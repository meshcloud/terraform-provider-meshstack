package clientmock

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshBuildingBlockRunnerClient struct {
	Store *Store[client.MeshBuildingBlockRunner]
}

func (m MeshBuildingBlockRunnerClient) Create(_ context.Context, runner client.MeshBuildingBlockRunner) (*client.MeshBuildingBlockRunner, error) {
	runnerUuid := uuid.NewString()
	restriction := runner.Spec.Restriction
	if restriction == nil {
		restriction = new("PRIVATE")
	}
	created := &client.MeshBuildingBlockRunner{
		Metadata: client.MeshBuildingBlockRunnerMetadata{
			Uuid:             new(runnerUuid),
			OwnedByWorkspace: runner.Metadata.OwnedByWorkspace,
			CreatedOn:        new("2026-01-01T00:00:00Z"),
			LastSeen:         new("2026-01-01T00:00:00Z"),
		},
		Spec: client.MeshBuildingBlockRunnerSpec{
			DisplayName:                runner.Spec.DisplayName,
			PublicKey:                  runner.Spec.PublicKey,
			ImplementationType:         runner.Spec.ImplementationType,
			Restriction:                restriction,
			IsSelfHosted:               new(true),
			WorkloadIdentityFederation: runner.Spec.WorkloadIdentityFederation,
		},
	}
	m.Store.Set(runnerUuid, created)
	return created, nil
}

func (m MeshBuildingBlockRunnerClient) Read(_ context.Context, uuid string) (*client.MeshBuildingBlockRunner, error) {
	if runner, ok := m.Store.Get(uuid); ok {
		return runner, nil
	}
	return nil, nil
}

func (m MeshBuildingBlockRunnerClient) Update(_ context.Context, runner client.MeshBuildingBlockRunner) (*client.MeshBuildingBlockRunner, error) {
	if runner.Metadata.Uuid == nil || *runner.Metadata.Uuid == "" {
		return nil, fmt.Errorf("building block runner uuid is required for update")
	}

	uuid := *runner.Metadata.Uuid
	if existing, ok := m.Store.Get(uuid); ok {
		existing.Spec = runner.Spec
		if existing.Spec.Restriction == nil {
			restriction := "PRIVATE"
			existing.Spec.Restriction = new(restriction)
		}
		if existing.Spec.IsSelfHosted == nil {
			isSelfHosted := true
			existing.Spec.IsSelfHosted = new(isSelfHosted)
		}
		return existing, nil
	}

	return nil, fmt.Errorf("building block runner not found: %s", uuid)
}

func (m MeshBuildingBlockRunnerClient) Delete(_ context.Context, uuid string) error {
	m.Store.Delete(uuid)
	return nil
}
