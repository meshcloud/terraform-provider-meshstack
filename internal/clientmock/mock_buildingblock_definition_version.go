package clientmock

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/client/types/ptr"
)

type MeshBuildingBlockDefinitionVersionClient struct {
	Store Store[client.MeshBuildingBlockDefinitionVersion]
}

func (m MeshBuildingBlockDefinitionVersionClient) List(_ context.Context, buildingBlockDefinitionUuid string) ([]client.MeshBuildingBlockDefinitionVersion, error) {
	var result []client.MeshBuildingBlockDefinitionVersion
	for _, version := range m.Store {
		if version.Spec.BuildingBlockDefinitionRef.Uuid == buildingBlockDefinitionUuid {
			result = append(result, *version)
		}
	}
	return result, nil
}

func (m MeshBuildingBlockDefinitionVersionClient) Create(_ context.Context, ownedByWorkspace string, versionSpec client.MeshBuildingBlockDefinitionVersionSpec) (*client.MeshBuildingBlockDefinitionVersion, error) {
	nextNum := m.getNextVersionNumber()
	versionUuid := acctest.RandString(32)
	// Compute hashes for all secrets in the spec
	backendSecretBehavior(true, &versionSpec, nil)

	// Set version number if not already set
	if versionSpec.VersionNumber == nil {
		versionSpec.VersionNumber = ptr.To(int64(nextNum))
	}

	created := &client.MeshBuildingBlockDefinitionVersion{
		ApiVersion: "v1",
		Kind:       "meshBuildingBlockDefinitionVersion",
		Metadata: client.MeshBuildingBlockDefinitionVersionMetadata{
			Uuid:             versionUuid,
			OwnedByWorkspace: ownedByWorkspace,
			CreatedOn:        "2024-01-01T00:00:00Z",
		},
		Spec: versionSpec,
	}

	m.Store[versionUuid] = created
	return created, nil
}

func (m MeshBuildingBlockDefinitionVersionClient) Update(_ context.Context, uuid string, ownedByWorkspace string, versionSpec client.MeshBuildingBlockDefinitionVersionSpec) (*client.MeshBuildingBlockDefinitionVersion, error) {
	if existing, ok := m.Store[uuid]; ok {
		// Compute hashes for all secrets in the spec
		backendSecretBehavior(false, &versionSpec, &existing.Spec)
		if existing.Metadata.OwnedByWorkspace != ownedByWorkspace {
			return nil, fmt.Errorf("mismatching workspace ownership: %s (existing) != %s (expected)", existing.Metadata.OwnedByWorkspace, ownedByWorkspace)
		}
		existing.Spec = versionSpec
		return existing, nil
	}
	return nil, fmt.Errorf("building block definition version not found: %s", uuid)
}

func (m MeshBuildingBlockDefinitionVersionClient) getNextVersionNumber() int {
	maxNum := 0
	for _, v := range m.Store {
		if v.Spec.VersionNumber != nil {
			num := int(*v.Spec.VersionNumber)
			if num > maxNum {
				maxNum = num
			}
		}
	}
	return maxNum + 1
}
