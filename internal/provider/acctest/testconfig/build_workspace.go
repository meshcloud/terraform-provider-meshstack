package testconfig

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

// Workspace builds a workspace config with a randomized identifier and an empty inline tags map.
func Workspace(t *testing.T) (config Config, workspaceAddr Traversal) {
	t.Helper()
	name := "test-ws-" + acctest.RandString(8)
	return Resource{Name: "workspace"}.Config(t).WithFirstBlock(
		ExtractAddress(&workspaceAddr),
		Descend("metadata")(
			Descend("name")(SetString(name)),
			Descend("tags")(SetRawExpr(`{}`)),
		),
	), workspaceAddr
}

// WorkspaceWithoutTags builds a workspace config with a randomized identifier and no inline tags,
// suitable for use alongside dedicated meshstack_workspace_tag(s) resources.
//
// Note: tags = {} is still declared explicitly. The empty declaration primes reconcileTrackedTags
// with an empty tracked-key set so the workspace resource ignores tags managed by the dedicated
// tag resources on the same workspace. Without this, the null prior state would cause
// reconcileTrackedTags to return the full API tag superset as drift.
func WorkspaceWithoutTags(t *testing.T) (config Config, workspaceAddr Traversal) {
	t.Helper()
	name := "test-ws-" + acctest.RandString(8)
	return Resource{Name: "workspace"}.Config(t).WithFirstBlock(
		ExtractAddress(&workspaceAddr),
		Descend("metadata")(
			Descend("name")(SetString(name)),
			Descend("tags")(SetRawExpr(`{}`)),
		),
	), workspaceAddr
}
