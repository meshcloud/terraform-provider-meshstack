package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccPaymentMethodDataSource(t *testing.T) {
	paymentMethodConfig, paymentMethodAddr, workspaceAddr := testconfig.PaymentMethodAndWorkspace(t)

	dataSourceAddress := testconfig.Traversal{"data.meshstack_payment_method", "example"}
	config := testconfig.DataSource{Name: "payment_method"}.Config(t).WithFirstBlock(
		testconfig.Descend("metadata")(
			testconfig.Descend("name")(testconfig.SetAddr(paymentMethodAddr, "metadata", "name")),
			testconfig.Descend("owned_by_workspace")(testconfig.SetAddr(workspaceAddr, "metadata", "name")),
		)).
		Join(paymentMethodConfig)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("metadata").AtMapKey("name"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Payment Method")),
				},
			},
		},
	})
}
