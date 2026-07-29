package testconfig

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

// WorkspaceTag builds a meshstack_workspace_tag config wired to the given workspace address.
// It creates a single tag drawn from a TagDefinition, returning the resource address and tag key.
func WorkspaceTag(t *testing.T, workspaceAddr Traversal) (config Config, workspaceTagAddr Traversal, tagKey string) {
	t.Helper()
	tagConfig, tagDefinitionAddr, tagKey := TagDefinition(t, "meshWorkspace")
	return Resource{Name: "workspace_tag"}.Config(t).WithFirstBlock(
		// Rename the block per run so several tag resources can coexist in one config — the primary use
		// case for this resource. Callers address it through the returned traversal, not by name.
		RenameKey("workspace_tag_"+acctest.RandString(8)),
		ExtractAddress(&workspaceTagAddr),
		Descend("metadata", "workspace_identifier")(SetAddr(workspaceAddr, "metadata", "name")),
		Descend("metadata", "key")(SetAddr(tagDefinitionAddr, "spec", "key")),
		Descend("spec", "values")(SetRawExpr(`["12345"]`)),
	).Join(tagConfig), workspaceTagAddr, tagKey
}
