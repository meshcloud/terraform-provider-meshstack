package testconfig

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

func BuildProjectConfig(t *testing.T, workspaceAddr Traversal) (config Config, projectAddr Traversal) {
	t.Helper()
	projectName := "test-proj-" + acctest.RandString(8)
	tagConfig, tagAddr := BuildTagDefinitionConfig(t, "meshProject")
	return Resource{Name: "project"}.Config(t).WithFirstBlock(t,
		ExtractIdentifier(&projectAddr),
		OwnedByWorkspace(t, workspaceAddr),
		Traverse(t, "metadata")(Traverse(t, "name")(SetString(projectName))),
		Traverse(t, "spec")(
			Traverse(t, "payment_method_identifier")(RemoveKey()),
			Traverse(t, "tags")(SetRawExpr(fmt.Sprintf("{(%s) = [\"tag-value1\", \"tag-value2\", \"tag-valueN\"]}", tagAddr.Join("spec", "key")))),
		),
	).Join(tagConfig), projectAddr
}

func BuildProjectAndWorkspaceConfig(t *testing.T) (config Config, projectAddr Traversal, workspaceAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := BuildWorkspaceConfig(t)
	projectConfig, projectAddr := BuildProjectConfig(t, workspaceAddr)
	return projectConfig.Join(workspaceConfig), projectAddr, workspaceAddr
}
