package testconfig

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

// CustomPlatform builds a custom platform config with a new platform type, owned by the given workspace.
func CustomPlatform(t *testing.T, workspaceAddr Traversal) (config Config, platformAddr, platformTypeAddr Traversal) {
	t.Helper()
	platformTypeConfig, platformTypeAddr := PlatformType(t, workspaceAddr)
	platformSuffix := acctest.RandString(8)
	return Resource{Name: "platform", Suffix: "_08_custom"}.Config(t).WithFirstBlock(
		ExtractAddress(&platformAddr),
		OwnedByWorkspace(workspaceAddr),
		Descend("metadata", "name")(SetString("custom-"+platformSuffix)),
		Descend("spec", "config", "custom", "platform_type_ref")(SetAddr(platformTypeAddr, "ref")),
	).Join(platformTypeConfig), platformAddr, platformTypeAddr
}

// CustomPlatformAndWorkspace builds a custom platform with all prerequisites (workspace, platform type).
func CustomPlatformAndWorkspace(t *testing.T) (config Config, platformAddr, workspaceAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := Workspace(t)
	platformConfig, platformAddr, _ := CustomPlatform(t, workspaceAddr)
	return platformConfig.Join(workspaceConfig), platformAddr, workspaceAddr
}

// PlatformAndWorkspace builds a platform config from the example resource with the given suffix and with workspace config.
func PlatformAndWorkspace(t *testing.T, suffix string) (config Config, platformAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := Workspace(t)
	platformSuffix := acctest.RandString(8)
	return Resource{Name: "platform", Suffix: suffix}.Config(t).WithFirstBlock(
		ExtractAddress(&platformAddr),
		OwnedByWorkspace(workspaceAddr),
		Descend("metadata", "name")(SetString(fmt.Sprintf("my-platform-%s", platformSuffix))),
	).Join(workspaceConfig), platformAddr
}
