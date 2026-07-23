package provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// updateIntegrationDisplayName clones the config and replaces "Integration" with "Updated Integration"
// in the integration resource's spec.display_name.
// azureDevopsPatPath is a factory (fresh path per call) to work around the slice copy/clone bug
// in tfjsonpath.Path.AtMapKey. Mirrors aksSecretPath in the platform test.
func azureDevopsPatPath() tfjsonpath.Path {
	return tfjsonpath.New("spec").AtMapKey("config").AtMapKey("azuredevops").AtMapKey("personal_access_token")
}

func updateIntegrationDisplayName(t *testing.T, config testconfig.Config, originalName string) string {
	t.Helper()
	updatedName := strings.Replace(originalName, "Integration", "Updated Integration", 1)
	return config.WithFirstBlock(
		testconfig.Descend("spec", "display_name")(testconfig.SetString(updatedName))).String()
}

func TestAccIntegrationResource(t *testing.T) {
	t.Parallel()

	t.Run("01_github", func(t *testing.T) {
		config, resourceAddress := testconfig.Integration(t, "_01_github")
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
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
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("spec"), checkIntegrationSpec("01_github", "GitHub Integration")),
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("status"), checkIntegrationStatus()),
						xknownvalue.Ref(resourceAddress, "meshIntegration", &resourceUuid),
					},
				},
				{
					Config: updateIntegrationDisplayName(t, config, "GitHub Integration"),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("spec"), checkIntegrationSpec("01_github", "GitHub Updated Integration")),
						xknownvalue.Ref(resourceAddress, "meshIntegration", &resourceUuid),
					},
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName: resourceAddress.String(),
				},
			},
		})
	})

	t.Run("02_azure_devops", func(t *testing.T) {
		config, resourceAddress := testconfig.Integration(t, "_02_azure_devops")
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
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
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("spec"), checkIntegrationSpec("02_azure_devops", "Azure DevOps Integration")),
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("status"), checkIntegrationStatus()),
						xknownvalue.Ref(resourceAddress, "meshIntegration", &resourceUuid),
					},
				},
				{
					Config: updateIntegrationDisplayName(t, config, "Azure DevOps Integration"),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("spec"), checkIntegrationSpec("02_azure_devops", "Azure DevOps Updated Integration")),
						xknownvalue.Ref(resourceAddress, "meshIntegration", &resourceUuid),
					},
				},
				// A different value gives a different secret_version hash, which rotates secret_value.
				{
					Config: func() string {
						u := config.WithFirstBlock(
							testconfig.Descend("spec", "config", "azuredevops", "personal_access_token")(
								testconfig.SetRawExpr(`provider::meshstack::non_ephemeral_secret("updated-plaintext-secret")`),
							))
						return u.String()
					}(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress.String(), plancheck.ResourceActionUpdate),
							plancheck.ExpectKnownValue(resourceAddress.String(), azureDevopsPatPath().AtMapKey("secret_version"), knownvalue.StringExact("b889814ec3c1da42df5abf57be4e989de7411b326ba30050fea6366185c0e206")),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("metadata"), checkIntegrationMetadata()),
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("spec"), checkIntegrationSpec("02_azure_devops", "Azure DevOps Integration")),
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("status"), checkIntegrationStatus()),
						xknownvalue.Ref(resourceAddress, "meshIntegration", &resourceUuid),
					},
				},
				// On import the config wants secret_version as the value's sha256, but the backend
				// returns its own hash, so the two differ and the first plan sends the secret again.
				// That plan is expected, not drift, and matches the platform AKS example.
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName:       resourceAddress.String(),
					ExpectNonEmptyPlan: true,
					ImportPlanChecks: resource.ImportPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress.String(), plancheck.ResourceActionUpdate),
							plancheck.ExpectUnknownValue(resourceAddress.String(), azureDevopsPatPath().AtMapKey("secret_hash")),
							plancheck.ExpectKnownValue(resourceAddress.String(), azureDevopsPatPath().AtMapKey("secret_version"), knownvalue.StringExact("b889814ec3c1da42df5abf57be4e989de7411b326ba30050fea6366185c0e206")),
						},
					},
				},
			},
		})
	})

	t.Run("03_gitlab", func(t *testing.T) {
		config, resourceAddress := testconfig.Integration(t, "_03_gitlab")
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
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
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("spec"), checkIntegrationSpec("03_gitlab", "GitLab Integration")),
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("status"), checkIntegrationStatus()),
						xknownvalue.Ref(resourceAddress, "meshIntegration", &resourceUuid),
					},
				},
				{
					Config: updateIntegrationDisplayName(t, config, "GitLab Integration"),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("spec"), checkIntegrationSpec("03_gitlab", "GitLab Updated Integration")),
						xknownvalue.Ref(resourceAddress, "meshIntegration", &resourceUuid),
					},
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName: resourceAddress.String(),
				},
			},
		})
	})

	t.Run("04_entra_id", func(t *testing.T) {
		// Entra ID integrations can only be owned by the admin (partner) workspace, so own it by
		// the pre-seeded AdminWorkspaceIdentifier instead of a freshly created test workspace.
		config, resourceAddress := testconfig.IntegrationForWorkspace(t, "_04_entra_id", AdminWorkspaceIdentifier)
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
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
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("spec"), checkIntegrationSpec("04_entra_id", "Entra ID Integration")),
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("status"), checkIntegrationStatus()),
						xknownvalue.Ref(resourceAddress, "meshIntegration", &resourceUuid),
					},
				},
				{
					Config: updateIntegrationDisplayName(t, config, "Entra ID Integration"),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("spec"), checkIntegrationSpec("04_entra_id", "Entra ID Updated Integration")),
						xknownvalue.Ref(resourceAddress, "meshIntegration", &resourceUuid),
					},
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName: resourceAddress.String(),
				},
			},
		})
	})
}

func checkIntegrationMetadata() knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"uuid":               xknownvalue.NotEmptyString(),
		"owned_by_workspace": xknownvalue.NotEmptyString(),
	})
}

func checkIntegrationSpec(exampleSuffix string, displayName string) knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"display_name": knownvalue.StringExact(displayName),
		"config":       checkIntegrationConfig(exampleSuffix),
	})
}

func checkIntegrationConfig(exampleSuffix string) knownvalue.Check {
	switch exampleSuffix {
	case "01_github":
		return xknownvalue.MapExact(map[string]knownvalue.Check{
			"github": xknownvalue.MapExact(map[string]knownvalue.Check{
				"owner":    knownvalue.StringExact("my-org"),
				"base_url": knownvalue.StringExact("https://github.com"),
				"app_id":   knownvalue.StringExact("123456"),
				"app_private_key": xknownvalue.MapExact(map[string]knownvalue.Check{
					"secret_value":   knownvalue.Null(),
					"secret_hash":    xknownvalue.NotEmptyString(),
					"secret_version": xknownvalue.NotEmptyString(),
				}),
				"runner_ref": xknownvalue.MapExact(map[string]knownvalue.Check{
					"uuid": knownvalue.StringExact(SharedBuildingBlockRunnerUuid),
					"kind": knownvalue.StringExact("meshBuildingBlockRunner"),
				}),
			}),
			"azuredevops": knownvalue.Null(),
			"gitlab":      knownvalue.Null(),
			"entraid":     knownvalue.Null(),
		})
	case "02_azure_devops":
		return xknownvalue.MapExact(map[string]knownvalue.Check{
			"github": knownvalue.Null(),
			"azuredevops": xknownvalue.MapExact(map[string]knownvalue.Check{
				"base_url":     knownvalue.StringExact("https://dev.azure.com"),
				"organization": knownvalue.StringExact("my-organization"),
				"personal_access_token": xknownvalue.MapExact(map[string]knownvalue.Check{
					"secret_value":   knownvalue.Null(),
					"secret_hash":    xknownvalue.NotEmptyString(),
					"secret_version": xknownvalue.NotEmptyString(),
				}),
				"runner_ref": xknownvalue.MapExact(map[string]knownvalue.Check{
					"uuid": knownvalue.StringExact(SharedBuildingBlockRunnerUuid),
					"kind": knownvalue.StringExact("meshBuildingBlockRunner"),
				}),
			}),
			"gitlab":  knownvalue.Null(),
			"entraid": knownvalue.Null(),
		})
	case "03_gitlab":
		return xknownvalue.MapExact(map[string]knownvalue.Check{
			"github":      knownvalue.Null(),
			"azuredevops": knownvalue.Null(),
			"gitlab": xknownvalue.MapExact(map[string]knownvalue.Check{
				"base_url": knownvalue.StringExact("https://gitlab.com"),
				"runner_ref": xknownvalue.MapExact(map[string]knownvalue.Check{
					"uuid": knownvalue.StringExact(SharedBuildingBlockRunnerUuid),
					"kind": knownvalue.StringExact("meshBuildingBlockRunner"),
				}),
			}),
			"entraid": knownvalue.Null(),
		})
	case "04_entra_id":
		return xknownvalue.MapExact(map[string]knownvalue.Check{
			"github":      knownvalue.Null(),
			"azuredevops": knownvalue.Null(),
			"gitlab":      knownvalue.Null(),
			"entraid": xknownvalue.MapExact(map[string]knownvalue.Check{
				"tenant_id": knownvalue.StringExact("xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"),
				"client_id": knownvalue.StringExact("yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy"),
				"client_secret": xknownvalue.MapExact(map[string]knownvalue.Check{
					"secret_value":   knownvalue.Null(),
					"secret_hash":    xknownvalue.NotEmptyString(),
					"secret_version": xknownvalue.NotEmptyString(),
				}),
				"redirect_url": xknownvalue.NotEmptyString(),
			}),
		})
	default:
		panic("unknown example suffix: " + exampleSuffix)
	}
}

func checkIntegrationStatus() knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"is_built_in":                  knownvalue.Bool(false),
		"workload_identity_federation": knownvalue.Null(),
	})
}
