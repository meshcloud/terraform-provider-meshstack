package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/stretchr/testify/assert"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/clientmock"
)

func TestAccPlatformResource(t *testing.T) {
	runPlatformResourceTestCase(t)
}

func TestPlatformResource(t *testing.T) {
	// Run acceptance tests as unit tests with mock
	runPlatformResourceTestCase(t, SetupMockClient(func(t *testing.T, testCase *resource.TestCase, mockClient clientmock.Client) {
		t.Helper()
		testCase.Steps[0].PostApplyFunc = func() {
			assert.Len(t, mockClient.Platform.Store, 1)
		}
	}))
}

func runPlatformResourceTestCase(t *testing.T, modifiers ...ResourceTestCaseModifier) {
	t.Helper()
	var resourceAddress, platformName examples.Identifier
	config := PlatformResourceConfigForTest(&resourceAddress, &platformName)

	testCase := resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress.String(), plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("metadata"), checkPlatformMetadata(platformName.String())),
				},
			},
		},
	}
	ResourceTestCaseModifiers(modifiers).ApplyAndTest(t, testCase)
}

func PlatformResourceConfigForTest(resourceAddress, platformName *examples.Identifier) examples.Config {
	name := examples.Identifier{fmt.Sprintf("acctest-platform-%s", acctest.RandString(12))}
	if platformName != nil {
		*platformName = name
	}

	config := examples.Resource{Name: "platform"}.Config().
		OwnedByAdminWorkspace().
		ReplaceAll(`name               = "my-azure-platform"`, name.Format(`name = "%s"`))

	if resourceAddress != nil {
		config = config.SingleResourceAddress(resourceAddress)
	}

	return config
}

func checkPlatformMetadata(name string) knownvalue.Check {
	return knownvalue.MapExact(map[string]knownvalue.Check{
		"name":               knownvalue.StringExact(name),
		"owned_by_workspace": knownvalue.StringExact("managed-customer"),
		"uuid":               KnownValueNotEmptyString(),
	})
}
