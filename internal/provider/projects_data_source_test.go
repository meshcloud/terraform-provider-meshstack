package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/meshcloud/meshstack-cli/client"
)

func TestAccProjectsDatasource(t *testing.T) {
	suffix := acctest.RandString(8)
	vars := hclStepVars{"suffix": suffix}

	project := func(name string, paymentMethodIdentifier, paymentMethodRef knownvalue.Check) knownvalue.Check {
		return knownvalue.ObjectPartial(map[string]knownvalue.Check{
			"ref": knownvalue.ObjectExact(map[string]knownvalue.Check{
				"kind":               knownvalue.StringExact(client.MeshObjectKind.Project),
				"owned_by_workspace": knownvalue.StringExact("test-ws-" + suffix),
				"name":               knownvalue.StringExact(name),
			}),
			"metadata": knownvalue.ObjectPartial(map[string]knownvalue.Check{
				"name":               knownvalue.StringExact(name),
				"owned_by_workspace": knownvalue.StringExact("test-ws-" + suffix),
			}),
			"spec": knownvalue.ObjectPartial(map[string]knownvalue.Check{
				"payment_method_identifier": paymentMethodIdentifier,
				"payment_method_ref":        paymentMethodRef,
			}),
		})
	}
	withPaymentMethod := project("test-proj-pm-"+suffix, knownvalue.StringExact("test-pm-"+suffix), knownvalue.ObjectExact(map[string]knownvalue.Check{
		"kind": knownvalue.StringExact(client.MeshObjectKind.PaymentMethod),
		"name": knownvalue.StringExact("test-pm-" + suffix),
	}))
	withoutPaymentMethod := project("test-proj-"+suffix, knownvalue.Null(), knownvalue.Null())

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "data-sources/projects", 1, vars),
			},
			{
				Config: hclStep(t, "data-sources/projects", 2, vars),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.meshstack_projects.example_all_projects_in_workspace", tfjsonpath.New("projects"),
						knownvalue.SetExact([]knownvalue.Check{withPaymentMethod, withoutPaymentMethod})),
				},
			},
			{
				Config: hclStep(t, "data-sources/projects", 3, vars),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.meshstack_projects.example_all_projects_in_workspace_with_payment_method", tfjsonpath.New("projects"),
						knownvalue.ListExact([]knownvalue.Check{withPaymentMethod})),
				},
			},
		},
	})
}
