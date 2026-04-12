package testconfig

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

func BuildLandingZoneConfig(t *testing.T, workspaceAddr, platformAddr, platformTypeAddr Traversal) (config Config, landingZoneAddr Traversal) {
	t.Helper()
	var bbdAddr Traversal
	bbdConfig := Resource{Name: "landingzone"}.TestSupportConfig(t, "_bbd").WithFirstBlock(t,
		ExtractIdentifier(&bbdAddr),
		OwnedByWorkspace(t, workspaceAddr),
		Traverse(t, "spec")(Traverse(t, "supported_platforms")(SetRawExpr(
			fmt.Sprintf("[{kind = \"meshPlatformType\", name = %s}]", platformTypeAddr.Join("metadata", "name")),
		))),
	)

	landingZoneSuffix := acctest.RandString(8)
	return Resource{Name: "landingzone", Suffix: "_02_custom"}.Config(t).WithFirstBlock(t,
		ExtractIdentifier(&landingZoneAddr),
		OwnedByWorkspace(t, workspaceAddr),
		Traverse(t, "metadata")(Traverse(t, "name")(SetString(fmt.Sprintf("test-lz-%s", landingZoneSuffix)))),
		Traverse(t, "spec")(
			Traverse(t, "platform_ref")(SetRawExpr(fmt.Sprintf("{uuid = %s}", platformAddr.Join("metadata", "uuid")))),
			Traverse(t, "mandatory_building_block_refs")(SetRawExpr(fmt.Sprintf("[{uuid = %s}]", bbdAddr.Join("metadata", "uuid")))),
		),
	).Join(bbdConfig), landingZoneAddr
}

func BuildLandingZoneAndWorkspaceConfig(t *testing.T) (config Config, landingZoneAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := BuildWorkspaceConfig(t)
	platformConfig, platformAddr, platformTypeAddr := BuildCustomPlatformConfig(t, workspaceAddr)
	landingZoneConfig, landingZoneAddr := BuildLandingZoneConfig(t, workspaceAddr, platformAddr, platformTypeAddr)
	return landingZoneConfig.Join(workspaceConfig, platformConfig), landingZoneAddr
}

func BuildSimpleLandingZoneConfig(t *testing.T, workspaceAddr, platformAddr Traversal) (config Config, landingZoneAddr Traversal) {
	t.Helper()
	landingZoneSuffix := acctest.RandString(8)
	return Resource{Name: "landingzone", Suffix: "_02_custom"}.Config(t).WithFirstBlock(t,
		ExtractIdentifier(&landingZoneAddr),
		OwnedByWorkspace(t, workspaceAddr),
		Traverse(t, "metadata")(Traverse(t, "name")(SetString(fmt.Sprintf("test-lz-%s", landingZoneSuffix)))),
		Traverse(t, "spec")(Traverse(t, "platform_ref")(SetRawExpr(fmt.Sprintf("{uuid = %s}", platformAddr.Join("metadata", "uuid"))))),
	), landingZoneAddr
}
