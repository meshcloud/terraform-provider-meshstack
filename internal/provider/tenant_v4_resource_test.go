package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccTenantV4(t *testing.T) {
	config, tenantAddr := testconfig.TenantV4AndWorkspace(t)
	resourceAddress := tenantAddr.String()

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					// Ref
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("ref").AtMapKey("kind"), knownvalue.StringExact("meshTenant")),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("ref").AtMapKey("uuid"), xknownvalue.NotEmptyString()),

					// Metadata
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata").AtMapKey("owned_by_project"), xknownvalue.NotEmptyString()),

					// Spec
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("platform_identifier"), xknownvalue.NotEmptyString()),
				},
			},
		},
	})
}
