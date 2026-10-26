package testconfig

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

// ProjectBinding builds a project binding of subjectKind "user" or "group" in a new project. The
// subject is bound to the workspace first, which meshStack requires.
func ProjectBinding(t *testing.T, subjectKind, subjectName string) (config Config, bindingAddr, workspaceAddr Traversal, bindingName string) {
	t.Helper()
	projectConfig, projectAddr, workspaceAddr := ProjectAndWorkspace(t)
	name := "project_" + subjectKind + "_binding"

	var workspaceBindingAddr Traversal
	workspaceBindingConfig := Resource{Name: name}.TestSupportConfig(t, "_workspace_binding").WithFirstBlock(
		ExtractAddress(&workspaceBindingAddr),
		Descend("metadata", "name")(SetString("test-wb-"+acctest.RandString(8))),
		Descend("target_ref", "name")(SetAddr(workspaceAddr, "metadata", "name")),
		Descend("subject", "name")(SetString(subjectName)),
	)

	bindingName = "test-pb-" + acctest.RandString(8)
	config = Resource{Name: name}.Config(t).WithFirstBlock(
		ExtractAddress(&bindingAddr),
		Descend("metadata", "name")(SetString(bindingName)),
		Descend("target_ref")(
			Descend("owned_by_workspace")(SetAddr(workspaceAddr, "metadata", "name")),
			Descend("name")(SetAddr(projectAddr, "metadata", "name")),
		),
		Descend("subject", "name")(SetAddr(workspaceBindingAddr, "subject", "name")),
	).Join(workspaceBindingConfig, projectConfig)
	return config, bindingAddr, workspaceAddr, bindingName
}
