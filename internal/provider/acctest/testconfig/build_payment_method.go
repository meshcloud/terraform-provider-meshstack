package testconfig

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

// PaymentMethod builds a payment method config owned by the given workspace.
func PaymentMethod(t *testing.T, workspaceAddr Traversal) (config Config, paymentMethodAddr Traversal) {
	t.Helper()
	tagConfig, tagDefinitionAddr, _ := TagDefinition(t, "meshPaymentMethod")
	paymentMethodName := "test-pm-" + acctest.RandString(8)
	return Resource{Name: "payment_method"}.Config(t).WithFirstBlock(
		ExtractAddress(&paymentMethodAddr),
		OwnedByWorkspace(workspaceAddr),
		Descend("metadata", "name")(SetString(paymentMethodName)),
		Descend("spec", "tags")(SetRawExpr(`{(%s) = ["0000"]}`, tagDefinitionAddr.Join("spec", "key"))),
	).Join(tagConfig), paymentMethodAddr
}

// PaymentMethodAndWorkspace builds a payment method config with a new workspace.
func PaymentMethodAndWorkspace(t *testing.T) (config Config, paymentMethodAddr, workspaceAddr Traversal) {
	t.Helper()
	workspaceConfig, workspaceAddr := Workspace(t)
	config, paymentMethodAddr = PaymentMethod(t, workspaceAddr)
	return config.Join(workspaceConfig), paymentMethodAddr, workspaceAddr
}
