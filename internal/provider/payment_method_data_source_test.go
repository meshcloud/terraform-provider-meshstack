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
	paymentMethodConfig, paymentMethodAddr, workspaceAddr := testconfig.BuildPaymentMethodConfig(t)

	dataSourceAddress := testconfig.Traversal{"data.meshstack_payment_method", "example"}
	dataSourceConfig := testconfig.DataSource{Name: "payment_method"}.Config(t)
	dataSourceConfig = dataSourceConfig.WithFirstBlock(t,
		testconfig.Traverse(t, "metadata")(
			testconfig.Traverse(t, "name")(testconfig.SetRawExpr(paymentMethodAddr.Format("%s.metadata.name"))),
			testconfig.Traverse(t, "owned_by_workspace")(testconfig.SetRawExpr(workspaceAddr.Format("%s.metadata.name"))),
		))

	config := dataSourceConfig.Join(paymentMethodConfig)

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
