package testconfig

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

// LandingZone builds a landing zone config wired to the given workspace, platform, and platform type.
func LandingZone(t *testing.T, workspaceAddr, platformAddr, platformTypeAddr Traversal) (config Config, landingZoneAddr Traversal) {
	t.Helper()
	var buildingBlockDefinitionAddr Traversal
	buildingBlockDefinitionConfig := Resource{Name: "landingzone"}.TestSupportConfig(t, "_bbd").WithFirstBlock(
		ExtractAddress(&buildingBlockDefinitionAddr),
		OwnedByWorkspace(workspaceAddr),
		Descend("spec", "supported_platforms")(SetRawExpr(
			`[{kind = "meshPlatformType", name = %s}]`, platformTypeAddr.Join("metadata", "name"),
		)),
	)

	landingZoneSuffix := acctest.RandString(8)
	return Resource{Name: "landingzone", Suffix: "_02_custom"}.Config(t).WithFirstBlock(
		ExtractAddress(&landingZoneAddr),
		OwnedByWorkspace(workspaceAddr),
		Descend("metadata", "name")(SetString(fmt.Sprintf("test-lz-%s", landingZoneSuffix))),
		Descend("spec")(
			Descend("platform_ref")(SetRawExpr("{uuid = %s}", platformAddr.Join("metadata", "uuid"))),
			Descend("mandatory_building_block_refs")(SetRawExpr("[{uuid = %s}]", buildingBlockDefinitionAddr.Join("metadata", "uuid"))),
		),
	).Join(buildingBlockDefinitionConfig), landingZoneAddr
}

// LandingZoneAndWorkspace builds a landing zone with all prerequisites (workspace, platform, platform type).
func LandingZoneAndWorkspace(t *testing.T) (config Config, landingZoneAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := Workspace(t)
	platformConfig, platformAddr, platformTypeAddr := CustomPlatform(t, workspaceAddr)
	landingZoneConfig, landingZoneAddr := LandingZone(t, workspaceAddr, platformAddr, platformTypeAddr)
	return landingZoneConfig.Join(workspaceConfig, platformConfig), landingZoneAddr
}

// SimpleLandingZone builds a landing zone with minimal config, without creating a platform type.
func SimpleLandingZone(t *testing.T, workspaceAddr, platformAddr Traversal) (config Config, landingZoneAddr Traversal) {
	t.Helper()
	landingZoneSuffix := acctest.RandString(8)
	return Resource{Name: "landingzone", Suffix: "_02_custom"}.Config(t).WithFirstBlock(
		ExtractAddress(&landingZoneAddr),
		OwnedByWorkspace(workspaceAddr),
		Descend("metadata", "name")(SetString(fmt.Sprintf("test-lz-%s", landingZoneSuffix))),
		Descend("spec", "platform_ref")(SetRawExpr("{uuid = %s}", platformAddr.Join("metadata", "uuid"))),
	), landingZoneAddr
}
