package testconfig

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

func BuildPlatformTypeConfig(t *testing.T, workspaceAddr Traversal) (config Config, platformTypeAddr Traversal) {
	t.Helper()
	platformTypeSuffix := strings.ToUpper(acctest.RandString(8))
	platformTypeName := "CUSTOM-PT-" + platformTypeSuffix
	config = Resource{Name: "platform_type"}.Config(t).WithFirstBlock(t,
		ExtractIdentifier(&platformTypeAddr),
		OwnedByWorkspace(t, workspaceAddr),
		Traverse(t, "metadata")(Traverse(t, "name")(SetString(platformTypeName))),
		Traverse(t, "spec")(Traverse(t, "display_name")(SetString("My Custom Platform "+platformTypeSuffix))),
	)
	return config, platformTypeAddr
}

func BuildPlatformTypeAndWorkspaceConfig(t *testing.T) (config Config, platformTypeAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := BuildWorkspaceConfig(t)
	platformTypeConfig, platformTypeAddr := BuildPlatformTypeConfig(t, workspaceAddr)
	return platformTypeConfig.Join(workspaceConfig), platformTypeAddr
}
