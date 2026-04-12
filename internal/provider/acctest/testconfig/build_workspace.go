package testconfig

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

func BuildWorkspaceConfig(t *testing.T) (config Config, workspaceAddr Traversal) {
	t.Helper()
	name := "test-ws-" + acctest.RandString(8)
	tagConfig, tagAddr := BuildTagDefinitionConfig(t, "meshWorkspace")
	return Resource{Name: "workspace"}.Config(t).WithFirstBlock(t,
		ExtractIdentifier(&workspaceAddr),
		Traverse(t, "metadata")(
			Traverse(t, "name")(SetString(name)),
			Traverse(t, "tags")(SetRawExpr(fmt.Sprintf("{(%s) = [\"12345\"]}", tagAddr.Join("spec", "key")))),
		),
	).Join(tagConfig), workspaceAddr
}
