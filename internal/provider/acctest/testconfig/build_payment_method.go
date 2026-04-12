package testconfig

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

func BuildPaymentMethodConfig(t *testing.T) (config Config, paymentMethodAddr Traversal, workspaceAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := BuildWorkspaceConfig(t)
	tagConfig, tagAddr := BuildTagDefinitionConfig(t, "meshPaymentMethod")
	pmName := "test-pm-" + acctest.RandString(8)
	return Resource{Name: "payment_method"}.Config(t).WithFirstBlock(t,
		ExtractIdentifier(&paymentMethodAddr),
		OwnedByWorkspace(t, workspaceAddr),
		Traverse(t, "metadata")(Traverse(t, "name")(SetString(pmName))),
		Traverse(t, "spec")(Traverse(t, "tags")(SetRawExpr(fmt.Sprintf("{(%s) = [\"0000\"]}", tagAddr.Join("spec", "key"))))),
	).Join(workspaceConfig, tagConfig), paymentMethodAddr, workspaceAddr
}
