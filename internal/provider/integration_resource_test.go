package provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/stretchr/testify/assert"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/clientmock"
)

func TestAccIntegrationResource(t *testing.T) {
	runIntegrationTestCases(t)
}

func TestIntegrationResource(t *testing.T) {
	runIntegrationTestCases(t, SetupMockClient(func(t *testing.T, testCase *resource.TestCase, mockClient clientmock.Client) {
		t.Helper()
		testCase.Steps[0].PostApplyFunc = func() {
			assert.Len(t, mockClient.Integration.Store, 1)
		}
	}))
}

func runIntegrationTestCases(t *testing.T, modifiers ...ResourceTestCaseModifier) {
	t.Helper()
	for exampleResource := range (examples.Resource{Name: "integration"}).All() {
		exampleSuffix := strings.TrimPrefix(exampleResource.Suffix, "_")
		t.Run(exampleSuffix, func(t *testing.T) {
			t.Parallel()
			var resourceAddress examples.Identifier
			config := exampleResource.Config().SingleResourceAddress(&resourceAddress)

			type DisplayName struct {
				Value        string
				UpdatedValue string
			}

			var displayNamesByExample = map[string]DisplayName{
				"01_github": {
					Value:        "GitHub Integration",
					UpdatedValue: "GitHub Updated Integration",
				},
				"02_azure_devops": {
					Value:        "Azure DevOps Integration",
					UpdatedValue: "Azure DevOps Updated Integration",
				},
				"03_gitlab": {
					Value:        "GitLab Integration",
					UpdatedValue: "GitLab Updated Integration",
				},
			}

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
							statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("metadata"), checkIntegrationMetadata()),
							statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("spec"), checkIntegrationSpec(exampleSuffix, displayNamesByExample[exampleSuffix].Value)),
							statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("status"), checkIntegrationStatus()),
						},
					},
					{
						Config: config.ReplaceAll(` Integration"`, ` Updated Integration"`).String(),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{
								plancheck.ExpectResourceAction(resourceAddress.String(), plancheck.ResourceActionUpdate),
							},
						},
						ConfigStateChecks: []statecheck.StateCheck{
							statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("spec"), checkIntegrationSpec(exampleSuffix, displayNamesByExample[exampleSuffix].UpdatedValue)),
						},
					},
				},
			}

			ResourceTestCaseModifiers(modifiers).ApplyAndTest(t, testCase)
		})
	}
}

func checkIntegrationMetadata() knownvalue.Check {
	return knownvalue.MapExact(map[string]knownvalue.Check{
		"uuid":               KnownValueNotEmptyString(),
		"owned_by_workspace": knownvalue.StringExact("my-workspace"),
	})
}

func checkIntegrationSpec(exampleSuffix string, displayName string) knownvalue.Check {
	return knownvalue.MapExact(map[string]knownvalue.Check{
		"display_name": knownvalue.StringExact(displayName),
		"config":       checkIntegrationConfig(exampleSuffix),
	})
}

func checkIntegrationConfig(exampleSuffix string) knownvalue.Check {
	switch exampleSuffix {
	case "01_github":
		return knownvalue.MapExact(map[string]knownvalue.Check{
			"github": knownvalue.MapExact(map[string]knownvalue.Check{
				"owner":           knownvalue.StringExact("my-org"),
				"base_url":        knownvalue.StringExact("https://github.com"),
				"app_id":          knownvalue.StringExact("123456"),
				"app_private_key": KnownValueNotEmptyString(),
				"runner_ref": knownvalue.MapExact(map[string]knownvalue.Check{
					"uuid": knownvalue.StringExact("dc8c57a1-823f-4e96-8582-0275fa27dc7b"),
					"kind": knownvalue.StringExact("meshBuildingBlockRunner"),
				}),
			}),
			"azuredevops": knownvalue.Null(),
			"gitlab":      knownvalue.Null(),
		})
	case "02_azure_devops":
		return knownvalue.MapExact(map[string]knownvalue.Check{
			"github": knownvalue.Null(),
			"azuredevops": knownvalue.MapExact(map[string]knownvalue.Check{
				"base_url":              knownvalue.StringExact("https://dev.azure.com"),
				"organization":          knownvalue.StringExact("my-organization"),
				"personal_access_token": KnownValueNotEmptyString(),
				"runner_ref": knownvalue.MapExact(map[string]knownvalue.Check{
					"uuid": knownvalue.StringExact("05cfa85f-2818-4bdd-b193-620e0187d7de"),
					"kind": knownvalue.StringExact("meshBuildingBlockRunner"),
				}),
			}),
			"gitlab": knownvalue.Null(),
		})
	case "03_gitlab":
		return knownvalue.MapExact(map[string]knownvalue.Check{
			"github":      knownvalue.Null(),
			"azuredevops": knownvalue.Null(),
			"gitlab": knownvalue.MapExact(map[string]knownvalue.Check{
				"base_url": knownvalue.StringExact("https://gitlab.com"),
				"runner_ref": knownvalue.MapExact(map[string]knownvalue.Check{
					"uuid": knownvalue.StringExact("f4f4402b-f54d-4ab9-93ae-c07e997041e9"),
					"kind": knownvalue.StringExact("meshBuildingBlockRunner"),
				}),
			}),
		})
	default:
		panic("unknown example suffix: " + exampleSuffix)
	}
}

func checkIntegrationStatus() knownvalue.Check {
	return knownvalue.MapExact(map[string]knownvalue.Check{
		"is_built_in": knownvalue.Bool(false),
		"workload_identity_federation": knownvalue.MapExact(map[string]knownvalue.Check{
			"issuer":  KnownValueNotEmptyString(),
			"subject": KnownValueNotEmptyString(),
			"gcp": knownvalue.MapExact(map[string]knownvalue.Check{
				"audience": KnownValueNotEmptyString(),
			}),
			"aws": knownvalue.MapExact(map[string]knownvalue.Check{
				"audience":   KnownValueNotEmptyString(),
				"thumbprint": KnownValueNotEmptyString(),
			}),
			"azure": knownvalue.MapExact(map[string]knownvalue.Check{
				"audience": KnownValueNotEmptyString(),
			}),
		}),
	})
}
