package testconfig

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

// Workspace builds a workspace config with a randomized identifier and one inline tag.
func Workspace(t *testing.T) (config Config, workspaceAddr Traversal) {
	t.Helper()
	name := "test-ws-" + acctest.RandString(8)
	tagConfig, tagDefinitionAddr, _ := TagDefinition(t, "meshWorkspace")
	return Resource{Name: "workspace"}.Config(t).WithFirstBlock(
		ExtractAddress(&workspaceAddr),
		Descend("metadata")(
			Descend("name")(SetString(name)),
			Descend("tags")(SetRawExpr(`{(%s) = ["12345"]}`, tagDefinitionAddr.Join("spec", "key"))),
		),
	).Join(tagConfig), workspaceAddr
}

// WorkspaceWithoutTags builds a workspace config with a randomized identifier and no inline tags,
// suitable for use alongside dedicated meshstack_workspace_tag(s) resources.
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
