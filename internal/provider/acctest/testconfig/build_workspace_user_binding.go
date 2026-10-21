package testconfig

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

// WorkspaceUserBinding binds the user to the workspace without an expiry date. meshStack binds a user
// to a project only when the user is bound to the project's workspace.
func WorkspaceUserBinding(t *testing.T, workspaceAddr Traversal, username string) (config Config, bindingAddr Traversal) {
	t.Helper()
	return Resource{Name: "workspace_user_binding"}.Config(t).WithFirstBlock(
		ExtractAddress(&bindingAddr),
		Descend("metadata", "name")(SetString("test-wub-"+acctest.RandString(8))),
		Descend("target_ref", "name")(SetAddr(workspaceAddr, "metadata", "name")),
		Descend("subject", "name")(SetString(username)),
		Descend("expiry_date")(RemoveKey()),
	), bindingAddr
}
