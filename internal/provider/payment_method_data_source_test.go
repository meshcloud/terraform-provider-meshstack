package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/meshcloud/meshstack-cli/client"
)

func TestAccPaymentMethodDataSource(t *testing.T) {
	suffix := acctest.RandString(8)
	vars := hclStepVars{"suffix": suffix}
	paymentMethodAddr := "meshstack_payment_method.test"
	dataSourceAddress := "data.meshstack_payment_method.example"

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "data-sources/payment_method", 1, vars),
			},
			{
				Config: hclStep(t, "data-sources/payment_method", 2, vars),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact("test-pm-"+suffix)),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Payment Method")),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("ref").AtMapKey("kind"), knownvalue.StringExact(client.MeshObjectKind.PaymentMethod)),
					statecheck.CompareValuePairs(dataSourceAddress, tfjsonpath.New("ref"), paymentMethodAddr, tfjsonpath.New("ref"), compare.ValuesSame()),
				},
			},
		},
	})
}
