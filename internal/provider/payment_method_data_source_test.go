package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccPaymentMethodDataSource(t *testing.T) {
	config := examples.JoinTestStepConfigs(
		examples.DataSource.TestStepConfig(t, "payment_method", 1),
		paymentMethodStepConfig(t, 1, "prerequisites"),
	)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:          config,
				ConfigVariables: SuffixVariables(acctest.RandString(8)),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(paymentMethodDataSourceAddr, tfjsonpath.New("metadata").AtMapKey("name"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(paymentMethodDataSourceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Payment Method")),
				},
			},
		},
	})
}
