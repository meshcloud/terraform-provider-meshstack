package provider

import (
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
)

// seededUserBindingCreateImportSteps returns create + import steps for a user binding whose subject is SeededUsername.
func seededUserBindingCreateImportSteps(config testconfig.Config, resourceAddress testconfig.Traversal, bindingName string, stateChecks ...statecheck.StateCheck) []resource.TestStep {
	return []resource.TestStep{
		{
			Config: config.String(),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(resourceAddress.String(), plancheck.ResourceActionCreate),
				},
			},
			ConfigStateChecks: append(
				[]statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact(bindingName)),
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("subject").AtMapKey("name"), knownvalue.StringExact(SeededUsername)),
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("status").AtMapKey("user_disabled"), knownvalue.Bool(false)),
				},
				stateChecks...,
			),
		},
		{
			ResourceName:    resourceAddress.String(),
			ImportState:     true,
			ImportStateId:   bindingName,
			ImportStateKind: resource.ImportBlockWithID,
		},
	}
}
