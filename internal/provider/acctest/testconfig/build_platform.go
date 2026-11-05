package testconfig

import (
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
