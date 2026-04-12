package testconfig

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

func BuildCustomPlatformConfig(t *testing.T, workspaceAddr Traversal) (config Config, platformAddr Traversal, platformTypeAddr Traversal) {
	t.Helper()
	platformTypeConfig, platformTypeAddr := BuildPlatformTypeConfig(t, workspaceAddr)
	platformSuffix := acctest.RandString(8)
	return Resource{Name: "platform", Suffix: "_08_custom"}.Config(t).WithFirstBlock(t,
		ExtractIdentifier(&platformAddr),
		OwnedByWorkspace(t, workspaceAddr),
		Traverse(t, "metadata")(Traverse(t, "name")(SetString("custom-"+platformSuffix))),
		Traverse(t, "spec", "config", "custom")(
			Traverse(t, "platform_type_ref")(SetRawExpr(platformTypeAddr.Join("ref").String())),
		),
	).Join(platformTypeConfig), platformAddr, platformTypeAddr
}

func BuildCustomPlatformAndWorkspaceConfig(t *testing.T) (config Config, platformAddr Traversal, workspaceAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := BuildWorkspaceConfig(t)
	platformConfig, platformAddr, _ := BuildCustomPlatformConfig(t, workspaceAddr)
	return platformConfig.Join(workspaceConfig), platformAddr, workspaceAddr
}

func BuildPlatformConfig(t *testing.T, suffix string) (config Config, platformAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := BuildWorkspaceConfig(t)
	platformSuffix := acctest.RandString(8)
	return Resource{Name: "platform", Suffix: suffix}.Config(t).WithFirstBlock(t,
		ExtractIdentifier(&platformAddr),
		OwnedByWorkspace(t, workspaceAddr),
		Traverse(t, "metadata")(Traverse(t, "name")(SetString(fmt.Sprintf("my-platform-%s", platformSuffix)))),
		TraverseAttributes(t)(SetBoolTrueIfFalse()),
	).Join(workspaceConfig), platformAddr
}
