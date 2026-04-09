package client

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

func TestKind(t *testing.T) {
	// verify hardcoded kind strings match InferKind for all client types
	assert.Equal(t, internal.InferKind[MeshBuildingBlock](), MeshObjectKind.BuildingBlock)
	assert.Equal(t, internal.InferKind[MeshBuildingBlockV2](), MeshObjectKind.BuildingBlock)
	assert.Equal(t, internal.InferKind[MeshBuildingBlockDefinition](), MeshObjectKind.BuildingBlockDefinition)
	assert.Equal(t, internal.InferKind[MeshBuildingBlockDefinitionVersion](), MeshObjectKind.BuildingBlockDefinitionVersion)
	assert.Equal(t, internal.InferKind[MeshIntegration](), MeshObjectKind.Integration)
	assert.Equal(t, internal.InferKind[MeshLandingZone](), MeshObjectKind.LandingZone)
	assert.Equal(t, internal.InferKind[MeshLocation](), MeshObjectKind.Location)
	assert.Equal(t, internal.InferKind[MeshPaymentMethod](), MeshObjectKind.PaymentMethod)
	assert.Equal(t, internal.InferKind[MeshPlatform](), MeshObjectKind.Platform)
	assert.Equal(t, internal.InferKind[MeshPlatformType](), MeshObjectKind.PlatformType)
	assert.Equal(t, internal.InferKind[MeshProject](), MeshObjectKind.Project)
	assert.Equal(t, internal.InferKind[MeshProjectGroupBinding](), MeshObjectKind.ProjectGroupBinding)
	assert.Equal(t, internal.InferKind[MeshProjectUserBinding](), MeshObjectKind.ProjectUserBinding)
	assert.Equal(t, internal.InferKind[MeshServiceInstance](), MeshObjectKind.ServiceInstance)
	assert.Equal(t, internal.InferKind[MeshTagDefinition](), MeshObjectKind.TagDefinition)
	assert.Equal(t, internal.InferKind[MeshTenant](), MeshObjectKind.Tenant)
	assert.Equal(t, internal.InferKind[MeshTenantV4](), MeshObjectKind.Tenant)
	assert.Equal(t, internal.InferKind[MeshWorkspace](), MeshObjectKind.Workspace)
	assert.Equal(t, internal.InferKind[MeshWorkspaceGroupBinding](), MeshObjectKind.WorkspaceGroupBinding)
	assert.Equal(t, internal.InferKind[MeshWorkspaceUserBinding](), MeshObjectKind.WorkspaceUserBinding)
}
