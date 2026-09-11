package clientmock

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"sync/atomic"

	"github.com/google/uuid"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type meshBuildingBlockDefinitionClient struct {
	Store        *Store[client.MeshBuildingBlockDefinition]
	StoreVersion *Store[client.MeshBuildingBlockDefinitionVersion]
	// RedactForNonOwnerAccess marks every definition redacted, as meshStack does for a workspace that
	// may consume a definition but does not own it. It only sets the mark; meshStack also strips the
	// implementation off the versions, which no caller reaches once the mark is read. Held behind a
	// pointer so a test can flip it between steps, after AsClient has copied this struct.
	RedactForNonOwnerAccess *atomic.Bool
}

func (m meshBuildingBlockDefinitionClient) List(_ context.Context, workspaceIdentifier *string) ([]client.MeshBuildingBlockDefinition, error) {
	var result []client.MeshBuildingBlockDefinition
	for _, def := range m.Store.Values() {
		if workspaceIdentifier == nil || def.Metadata.OwnedByWorkspace == *workspaceIdentifier {
			result = append(result, m.withStatus(*def))
		}
	}
	return result, nil
}

func (m meshBuildingBlockDefinitionClient) Read(_ context.Context, uuid string) (*client.MeshBuildingBlockDefinition, error) {
	if def, ok := m.Store.Get(uuid); ok {
		return new(m.withStatus(*def)), nil
	}
	return nil, nil
}

func (m meshBuildingBlockDefinitionClient) Create(_ context.Context, definition client.MeshBuildingBlockDefinition) (*client.MeshBuildingBlockDefinition, error) {
	definitionUuid := uuid.NewString()
	definition.Metadata.Uuid = new(definitionUuid)
	if definition.Spec.Symbol == nil {
		definition.Spec.Symbol = new("mock-default-symbol")
	}
	m.Store.Set(definitionUuid, &definition)

	// Create initial empty version (as the backend does)
	versionUuid := uuid.NewString()
	m.StoreVersion.Set(versionUuid, &client.MeshBuildingBlockDefinitionVersion{
		Metadata: client.MeshBuildingBlockDefinitionVersionMetadata{
			Uuid:             versionUuid,
			OwnedByWorkspace: definition.Metadata.OwnedByWorkspace,
		},
		Spec: client.MeshBuildingBlockDefinitionVersionSpec{
			BuildingBlockDefinitionRef: &client.UuidRef{
				Uuid: definitionUuid,
				Kind: "meshBuildingBlockDefinition",
			},
			DeletionMode:  client.BuildingBlockDeletionModeDelete.Unwrap(),
			VersionNumber: new(int64(1)),
			State:         client.MeshBuildingBlockDefinitionVersionStateDraft.Ptr(),
		},
	})
	return new(m.withStatus(definition)), nil
}

func (m meshBuildingBlockDefinitionClient) Update(_ context.Context, uuid string, definition client.MeshBuildingBlockDefinition) (*client.MeshBuildingBlockDefinition, error) {
	if existing, ok := m.Store.Get(uuid); ok {
		existing.Spec = definition.Spec
		existing.Metadata.Tags = definition.Metadata.Tags
		return new(m.withStatus(*existing)), nil
	}
	return nil, fmt.Errorf("building block definition not found: %s", uuid)
}

func (m meshBuildingBlockDefinitionClient) Delete(_ context.Context, uuid string) error {
	m.Store.Delete(uuid)
	return nil
}

// withStatus derives the definition status from the stored versions, which is what the backend sends
// on every definition it returns. The data source falls back to it whenever the version spec is not
// readable, so the mock has to answer with it for that fallback to be reachable at all.
func (m meshBuildingBlockDefinitionClient) withStatus(definition client.MeshBuildingBlockDefinition) client.MeshBuildingBlockDefinition {
	definitionUuid := ""
	if definition.Metadata.Uuid != nil {
		definitionUuid = *definition.Metadata.Uuid
	}

	var versions []client.MeshBuildingBlockDefinitionVersion
	for _, version := range m.StoreVersion.Values() {
		if version.Spec.BuildingBlockDefinitionRef != nil && version.Spec.BuildingBlockDefinitionRef.Uuid == definitionUuid &&
			version.Spec.VersionNumber != nil && version.Spec.State != nil {
			versions = append(versions, *version)
		}
	}
	slices.SortFunc(versions, func(a, b client.MeshBuildingBlockDefinitionVersion) int {
		return cmp.Compare(*a.Spec.VersionNumber, *b.Spec.VersionNumber)
	})

	status := client.MeshBuildingBlockDefinitionStatus{
		UsageCount:                new(int64(0)),
		RedactedForNonOwnerAccess: m.RedactForNonOwnerAccess.Load(),
	}
	for _, version := range versions {
		status.Versions = append(status.Versions, client.MeshBuildingBlockDefinitionStatusVersion{
			VersionUuid:   version.Metadata.Uuid,
			VersionNumber: *version.Spec.VersionNumber,
			State:         *version.Spec.State,
		})
		status.LatestVersion = *version.Spec.VersionNumber
		status.LatestVersionUuid = version.Metadata.Uuid
		if *version.Spec.State == client.MeshBuildingBlockDefinitionVersionStateReleased.Unwrap() {
			status.LatestReleasedVersion = new(*version.Spec.VersionNumber)
			status.LatestReleasedVersionUuid = new(version.Metadata.Uuid)
		}
	}

	definition.Status = &status
	return definition
}
