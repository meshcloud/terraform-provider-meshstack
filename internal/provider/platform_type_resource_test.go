package provider

import (
	_ "embed"
	"fmt"
	"strings"
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

func TestAccPlatformType(t *testing.T) {
	runPlatformTypeResourceTestCase(t)
}

func TestPlatformType(t *testing.T) {
	// Run acceptance tests as unit tests with mock
	runPlatformTypeResourceTestCase(t, SetupMockClient(func(t *testing.T, testCase *resource.TestCase, mockClient clientmock.Client) {
		t.Helper()
		testCase.Steps[0].PostApplyFunc = func() {
			assert.Len(t, mockClient.PlatformType.Store, 1)
		}
	}))
}

func PlatformTypeResourceConfigForTest(resourceAddress, platformTypeName *examples.Identifier, displayNameOut *string) examples.Config {
	name := examples.Identifier{strings.ToUpper("testacc-platform-type" + acctest.RandString(12))}
	if platformTypeName != nil {
		*platformTypeName = name
	}
	displayName := "Display name for " + name.String()
	if displayNameOut != nil {
		*displayNameOut = displayName
	}
	return examples.Resource{Name: "platform_type"}.Config().
		SingleResourceAddress(resourceAddress).
		OwnedByAdminWorkspace().
		ReplaceAll(`name               = "MY-PLATFORM-TYPE"`, name.Format(`name = "%s"`)).
		ReplaceAll(`display_name     = "My Custom Platform"`, fmt.Sprintf(`display_name = "%s"`, displayName))
}

func runPlatformTypeResourceTestCase(t *testing.T, modifiers ...ResourceTestCaseModifier) {
	t.Helper()
	var resourceAddress, platformTypeName examples.Identifier
	var displayName string
	config := PlatformTypeResourceConfigForTest(&resourceAddress, &platformTypeName, &displayName)
	updatedDisplayName := displayName + "-updated"

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
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("metadata"), checkPlatformTypeMetadata(platformTypeName.String())),
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("spec"), checkPlatformTypeSpec(displayName)),
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("status"), checkPlatformTypeStatus()),
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("ref"), checkPlatformTypeRef(platformTypeName.String())),
				},
			},
			{
				Config: config.ReplaceAll(displayName, updatedDisplayName).String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress.String(), plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("spec"), checkPlatformTypeSpec(updatedDisplayName)),
				},
			},
			{
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ImportStateId:   platformTypeName.String(),
				ResourceName:    resourceAddress.String(),
			},
		},
	}

	ResourceTestCaseModifiers(modifiers).ApplyAndTest(t, testCase)
}

func checkPlatformTypeMetadata(name string) knownvalue.Check {
	return knownvalue.MapExact(map[string]knownvalue.Check{
		"name":               knownvalue.StringExact(name),
		"owned_by_workspace": knownvalue.StringExact("managed-customer"),
		"uuid":               KnownValueNotEmptyString(),
	})
}

func checkPlatformTypeSpec(displayName string) knownvalue.Check {
	return knownvalue.MapExact(map[string]knownvalue.Check{
		"display_name":     knownvalue.StringExact(displayName),
		"category":         knownvalue.StringExact("CUSTOM"),
		"default_endpoint": knownvalue.StringExact("https://platform.example.com"),
		"icon":             knownvalue.StringExact("data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciLz4="),
	})
}

func checkPlatformTypeStatus() knownvalue.Check {
	return knownvalue.MapExact(map[string]knownvalue.Check{
		"lifecycle": knownvalue.MapExact(map[string]knownvalue.Check{
			"state": knownvalue.StringExact("ACTIVE"),
		}),
	})
}

func checkPlatformTypeRef(name string) knownvalue.Check {
	return knownvalue.MapExact(map[string]knownvalue.Check{
		"kind": knownvalue.StringExact("meshPlatformType"),
		"name": knownvalue.StringExact(name),
	})
}
