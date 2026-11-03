package testconfig

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

// Project builds a project config owned by the given workspace.
func Project(t *testing.T, workspaceAddr Traversal) (config Config, projectAddr Traversal) {
	t.Helper()
	projectName := "test-proj-" + acctest.RandString(8)
	tagConfig, tagDefinitionAddr, _ := TagDefinition(t, "meshProject")
	paymentMethodConfig, paymentMethodAddr := PaymentMethod(t, workspaceAddr)
	return Resource{Name: "project"}.Config(t).WithFirstBlock(
		ExtractAddress(&projectAddr),
		OwnedByWorkspace(workspaceAddr),
		Descend("metadata", "name")(SetString(projectName)),
		Descend("spec")(
			Descend("payment_method_identifier")(SetAddr(paymentMethodAddr, "metadata", "name")),
			Descend("tags")(SetRawExpr(`{(%s) = ["tag-value1", "tag-value2", "tag-valueN"]}`, tagDefinitionAddr.Join("spec", "key"))),
		),
	).Join(tagConfig, paymentMethodConfig), projectAddr
}
