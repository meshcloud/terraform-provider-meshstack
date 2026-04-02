package provider

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/zclconf/go-cty/cty"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/client/types/enum"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func updateBBDDescription(t *testing.T, config testconfig.Config, newDescription string) testconfig.Config {
	t.Helper()
	return config.WithFirstBlock(
		testconfig.Descend("spec", "description")(testconfig.SetString(newDescription)),
	)
}

func releaseBBDVersion(t *testing.T, config testconfig.Config) testconfig.Config {
	t.Helper()
	return config.WithFirstBlock(
		testconfig.Descend("version_spec", "draft")(testconfig.SetValue(cty.False)),
	)
}

func TestAccBuildingBlockDefinition(t *testing.T) {
	t.Parallel()

	var (
		versionStateDraft    = client.MeshBuildingBlockDefinitionVersionStateDraft
		versionStateReleased = client.MeshBuildingBlockDefinitionVersionStateReleased
	)

	expectedVersion := func(number int64, state enum.Entry[client.MeshBuildingBlockDefinitionVersionState]) knownvalue.Check {
		return xknownvalue.MapExact(map[string]knownvalue.Check{
			"uuid":         xknownvalue.NotEmptyString(),
			"number":       knownvalue.Int64Exact(number),
			"state":        knownvalue.StringExact(state.String()),
			"content_hash": xknownvalue.NotEmptyString(),
		})
	}

	const bbdDescription = "An example building block definition"

	t.Run("01_terraform", func(t *testing.T) {
		config, addr := testconfig.BBDTerraform(t)
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("metadata"), checkBBDMetadataFull()),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("spec"), checkBBDSpecFull(bbdDescription)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec("01_terraform", versionStateDraft, 1)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest_release"), knownvalue.Null()),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{expectedVersion(1, versionStateDraft)})),
						xknownvalue.Ref(addr, "meshBuildingBlockDefinition", &resourceUuid),
					},
				},
				// Step 2: Change secret input name (remove/add operation on inputs map)
				{
					Config: func() string {
						u := config.WithFirstBlock(
							testconfig.Descend("version_spec", "inputs", "SOMETHING_VERY_SECRET")(testconfig.RenameKey("SOMETHING_VERY_SECRET_RENAMED")))
						return u.String()
					}(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr.String(), plancheck.ResourceActionUpdate),
						},
					},
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName: addr.String(),
				},
			},
		})
	})

	t.Run("02_github_workflows", func(t *testing.T) {
		config, addr := testconfig.BBDWithIntegration(t, "02_github_workflows")
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("metadata"), checkBBDMetadataMinimal()),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("spec"), checkBBDSpecMinimal(bbdDescription)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec("02_github_workflows", versionStateDraft, 1)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest_release"), knownvalue.Null()),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{expectedVersion(1, versionStateDraft)})),
						xknownvalue.Ref(addr, "meshBuildingBlockDefinition", &resourceUuid),
					},
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName: addr.String(),
				},
			},
		})
	})

	t.Run("03_manual", func(t *testing.T) {
		config, addr := testconfig.BBDManual(t)
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Step 1: Create
				{
					Config: config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("metadata"), checkBBDMetadataMinimal()),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("spec"), checkBBDSpecMinimal(bbdDescription)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec("03_manual", versionStateDraft, 1)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest_release"), knownvalue.Null()),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{expectedVersion(1, versionStateDraft)})),
						xknownvalue.Ref(addr, "meshBuildingBlockDefinition", &resourceUuid),
					},
				},
				// Step 2: Update spec (description change, no new version)
				{
					Config: updateBBDDescription(t, config, "An updated building block definition").String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("metadata"), checkBBDMetadataMinimal()),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("spec"), checkBBDSpecMinimal("An updated building block definition")),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{expectedVersion(1, versionStateDraft)})),
						xknownvalue.Ref(addr, "meshBuildingBlockDefinition", &resourceUuid),
					},
				},
				// Step 3: Release (draft=false)
				{
					Config: releaseBBDVersion(t, config).String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec("03_manual", versionStateReleased, 1)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{expectedVersion(1, versionStateReleased)})),
					},
				},
				// Step 4: New draft (draft=true again, description changed)
				{
					Config: updateBBDDescription(t, config, "An updated building block definition").String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec("03_manual", versionStateDraft, 2)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest"), expectedVersion(2, versionStateDraft)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{
							expectedVersion(1, versionStateReleased),
							expectedVersion(2, versionStateDraft),
						})),
					},
				},
				// Step 5: Release the new draft (draft=false)
				{
					Config: releaseBBDVersion(t, config).String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec("03_manual", versionStateReleased, 2)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest_release"), expectedVersion(2, versionStateReleased)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest"), expectedVersion(2, versionStateReleased)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{
							expectedVersion(1, versionStateReleased),
							expectedVersion(2, versionStateReleased),
						})),
					},
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName: addr.String(),
				},
			},
		})
	})

	t.Run("04_azure_devops_pipeline", func(t *testing.T) {
		config, addr := testconfig.BBDWithIntegration(t, "04_azure_devops_pipeline")
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("metadata"), checkBBDMetadataMinimal()),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("spec"), checkBBDSpecMinimal(bbdDescription)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec("04_azure_devops_pipeline", versionStateDraft, 1)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest_release"), knownvalue.Null()),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{expectedVersion(1, versionStateDraft)})),
						xknownvalue.Ref(addr, "meshBuildingBlockDefinition", &resourceUuid),
					},
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName: addr.String(),
				},
			},
		})
	})

	t.Run("05_gitlab_pipeline", func(t *testing.T) {
		config, addr := testconfig.BBDGitlabPipeline(t)
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Step 1: Plan-only (ensure tf plan works before apply)
				{
					Config:             config.String(),
					PlanOnly:           true,
					ExpectNonEmptyPlan: true,
				},
				// Step 2: Create
				{
					Config: config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("metadata"), checkBBDMetadataMinimal()),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("spec"), checkBBDSpecMinimal(bbdDescription)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec("05_gitlab_pipeline", versionStateDraft, 1)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest_release"), knownvalue.Null()),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{expectedVersion(1, versionStateDraft)})),
						xknownvalue.Ref(addr, "meshBuildingBlockDefinition", &resourceUuid),
					},
				},
				// Step 3: Import
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName: addr.String(),
				},
				// Step 4: Rotate secret after import
				{
					Config: func() string {
						u := config.WithFirstBlock(
							testconfig.Descend("version_spec", "implementation", "gitlab_pipeline", "pipeline_trigger_token")(
								testconfig.Descend("secret_value")(testconfig.SetString("updated-plaintext-secret")),
								testconfig.Descend("secret_version")(testconfig.SetString("v1")),
							))
						return u.String()
					}(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("metadata"), checkBBDMetadataMinimal()),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("spec"), checkBBDSpecMinimal(bbdDescription)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{expectedVersion(1, versionStateDraft)})),
						xknownvalue.Ref(addr, "meshBuildingBlockDefinition", &resourceUuid),
					},
				},
			},
		})
	})
}

// checkBBDMetadataFull checks metadata for the 01_terraform example (tags with 2 entries).
func checkBBDMetadataFull() knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"uuid":               xknownvalue.NotEmptyString(),
		"owned_by_workspace": xknownvalue.NotEmptyString(),
		"tags":               knownvalue.MapSizeExact(2),
	})
}

// checkBBDMetadataMinimal checks metadata for examples without tags.
func checkBBDMetadataMinimal() knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"uuid":               xknownvalue.NotEmptyString(),
		"owned_by_workspace": xknownvalue.NotEmptyString(),
		"tags":               knownvalue.MapSizeExact(0),
	})
}

// checkBBDSpecFull checks spec for the 01_terraform example (all optional attributes set).
func checkBBDSpecFull(expectedDescription string) knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"display_name": knownvalue.StringFunc(func(v string) error {
			if !strings.HasPrefix(v, "Example Building Block") {
				return fmt.Errorf("expected %s to start with %s", v, "Example Building Block")
			}
			return nil
		}),
		"symbol": knownvalue.StringFunc(func(v string) error {
			if !strings.HasPrefix(v, "data:image/png;base64,") {
				return fmt.Errorf("value does not start with %s", "data:image/png;base64,")
			}
			return nil
		}),
		"description":       knownvalue.StringExact(expectedDescription),
		"readme":            xknownvalue.NotEmptyString(),
		"support_url":       knownvalue.StringExact("https://support.example.com/building-blocks"),
		"documentation_url": knownvalue.StringExact("https://docs.example.com/building-blocks"),
		"target_type":       knownvalue.StringExact("TENANT_LEVEL"),
		"supported_platforms": knownvalue.SetExact([]knownvalue.Check{
			xknownvalue.MapExact(map[string]knownvalue.Check{
				"kind": knownvalue.StringExact("meshPlatformType"),
				"name": knownvalue.StringExact("AZURE"),
			}),
			xknownvalue.MapExact(map[string]knownvalue.Check{
				"kind": knownvalue.StringExact("meshPlatformType"),
				"name": knownvalue.StringExact("AWS"),
			}),
		}),
		"run_transparency":          knownvalue.Bool(true),
		"use_in_landing_zones_only": knownvalue.Bool(true),
		"notification_subscribers": knownvalue.ListExact([]knownvalue.Check{
			knownvalue.StringExact("email:ops@example.com"),
		}),
	})
}

// checkBBDSpecMinimal checks spec for examples with only required attributes (workspace-level, no extras).
func checkBBDSpecMinimal(expectedDescription string) knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"display_name": knownvalue.StringFunc(func(v string) error {
			if !strings.HasPrefix(v, "Example Building Block") {
				return fmt.Errorf("expected %s to start with %s", v, "Example Building Block")
			}
			return nil
		}),
		"symbol":                    xknownvalue.NotEmptyString(),
		"description":               knownvalue.StringExact(expectedDescription),
		"readme":                    knownvalue.Null(),
		"support_url":               knownvalue.Null(),
		"documentation_url":         knownvalue.Null(),
		"target_type":               knownvalue.StringExact("WORKSPACE_LEVEL"),
		"supported_platforms":       knownvalue.Null(),
		"run_transparency":          knownvalue.Bool(false),
		"use_in_landing_zones_only": knownvalue.Bool(false),
		"notification_subscribers":  knownvalue.SetSizeExact(0),
	})
}

func checkBuildingBlockVersionSpec(exampleSuffix string, expectedState enum.Entry[client.MeshBuildingBlockDefinitionVersionState], expectedNumber int64) knownvalue.Check {
	exampleSuffixesToImplementationType := map[string]enum.Entry[client.MeshBuildingBlockImplementationType]{
		"01_terraform":             client.MeshBuildingBlockImplementationTypeTerraform,
		"02_github_workflows":      client.MeshBuildingBlockImplementationTypeGithubWorkflows,
		"03_manual":                client.MeshBuildingBlockImplementationTypeManual,
		"04_azure_devops_pipeline": client.MeshBuildingBlockImplementationTypeAzureDevOpsPipeline,
		"05_gitlab_pipeline":       client.MeshBuildingBlockImplementationTypeGitlabPipeline,
	}

	checkInputs, checkImplementation, checkOutputs := checksForImplementation(exampleSuffix)
	expectedDeletionMode := "DELETE"
	if exampleSuffix == "02_github_workflows" {
		expectedDeletionMode = "PURGE"
	}
	expected := map[string]knownvalue.Check{
		"state":                      knownvalue.StringExact(expectedState.String()),
		"version_number":             knownvalue.Int64Exact(expectedNumber),
		"draft":                      knownvalue.Bool(expectedState == client.MeshBuildingBlockDefinitionVersionStateDraft),
		"only_apply_once_per_tenant": knownvalue.Bool(exampleSuffix == "01_terraform"),
		"deletion_mode":              knownvalue.StringExact(expectedDeletionMode),
		"runner_ref": xknownvalue.MapExact(map[string]knownvalue.Check{
			"kind": knownvalue.StringExact("meshBuildingBlockRunner"),
			"uuid": knownvalue.StringExact(SharedBuildingBlockRunnerUuids[exampleSuffixesToImplementationType[exampleSuffix]]),
		}),
		"dependency_refs": knownvalue.SetSizeExact(0),
		"inputs":          checkInputs,
		"implementation":  checkImplementation,
		"outputs":         checkOutputs,
		"permissions":     knownvalue.SetSizeExact(0),
	}

	if exampleSuffix == "01_terraform" {
		expected["dependency_refs"] = knownvalue.ListExact([]knownvalue.Check{
			xknownvalue.MapExact(map[string]knownvalue.Check{
				"kind": knownvalue.StringExact("meshBuildingBlockDefinition"),
				"uuid": xknownvalue.NotEmptyString(),
			}),
		})
		expected["permissions"] = knownvalue.SetExact([]knownvalue.Check{
			knownvalue.StringExact("TENANT_SAVE"),
			knownvalue.StringExact("TENANT_LIST"),
		})
	}
	return xknownvalue.MapExact(expected)
}

func checksForImplementation(exampleSuffix string) (checkInputs, checkImplementation, checkOutputs knownvalue.Check) {
	switch exampleSuffix {
	case "01_terraform":
		return xknownvalue.MapExact(map[string]knownvalue.Check{
				"environment": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":           knownvalue.StringExact("Environment"),
					"type":                   knownvalue.StringExact("SINGLE_SELECT"),
					"assignment_type":        knownvalue.StringExact("USER_INPUT"),
					"is_environment":         knownvalue.Bool(false),
					"updateable_by_consumer": knownvalue.Bool(false),
					"description":            knownvalue.StringExact("The target environment"),
					"selectable_values": knownvalue.ListExact([]knownvalue.Check{
						knownvalue.StringExact("dev"),
						knownvalue.StringExact("prod"),
						knownvalue.StringExact("staging"),
					}),
					"value_validation_regex":         knownvalue.Null(),
					"validation_regex_error_message": knownvalue.Null(),
					"argument":                       knownvalue.Null(),
					"default_value":                  knownvalue.Null(),
					"sensitive":                      knownvalue.Null(),
				}),
				"resource_name": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":                   knownvalue.StringExact("Resource Name"),
					"type":                           knownvalue.StringExact("STRING"),
					"assignment_type":                knownvalue.StringExact("USER_INPUT"),
					"is_environment":                 knownvalue.Bool(false),
					"updateable_by_consumer":         knownvalue.Bool(true),
					"description":                    knownvalue.StringExact("Name of the resource to create"),
					"argument":                       knownvalue.Null(),
					"default_value":                  knownvalue.StringExact(`"some-resource-name"`),
					"value_validation_regex":         knownvalue.StringExact("^[a-z0-9-]+$"),
					"validation_regex_error_message": knownvalue.StringExact("Resource name must contain only lowercase letters, numbers, and hyphens"),
					"selectable_values":              knownvalue.Null(),
					"sensitive":                      knownvalue.Null(),
				}),
				"SOMETHING_VERY_SECRET": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":           knownvalue.StringExact("Top Secret"),
					"type":                   knownvalue.StringExact("STRING"),
					"assignment_type":        knownvalue.StringExact("STATIC"),
					"is_environment":         knownvalue.Bool(true),
					"updateable_by_consumer": knownvalue.Bool(false),
					"description":            knownvalue.StringExact("Really secret"),
					"sensitive": xknownvalue.MapExact(map[string]knownvalue.Check{
						"argument": xknownvalue.MapExact(map[string]knownvalue.Check{
							"secret_value":   knownvalue.Null(),
							"secret_hash":    xknownvalue.NotEmptyString(),
							"secret_version": xknownvalue.NotEmptyString(),
						}),
						"default_value": knownvalue.Null(),
					}),
					"value_validation_regex":         knownvalue.Null(),
					"validation_regex_error_message": knownvalue.Null(),
					"selectable_values":              knownvalue.Null(),
					"argument":                       knownvalue.Null(),
					"default_value":                  knownvalue.Null(),
				}),
				"some-file.yaml": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":                   knownvalue.StringExact("Some input file"),
					"type":                           knownvalue.StringExact("FILE"),
					"assignment_type":                knownvalue.StringExact("STATIC"),
					"is_environment":                 knownvalue.Bool(false),
					"updateable_by_consumer":         knownvalue.Bool(false),
					"description":                    knownvalue.Null(),
					"argument":                       xknownvalue.NotEmptyString(),
					"default_value":                  knownvalue.Null(),
					"value_validation_regex":         knownvalue.Null(),
					"validation_regex_error_message": knownvalue.Null(),
					"selectable_values":              knownvalue.Null(),
					"sensitive":                      knownvalue.Null(),
				}),
			}),
			xknownvalue.MapExact(map[string]knownvalue.Check{
				"manual":                knownvalue.Null(),
				"github_workflows":      knownvalue.Null(),
				"gitlab_pipeline":       knownvalue.Null(),
				"azure_devops_pipeline": knownvalue.Null(),
				"terraform":             checkTerraformImplementation(),
			}), xknownvalue.MapExact(map[string]knownvalue.Check{
				"some_output_flag": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":    knownvalue.StringExact("If true, it really worked"),
					"type":            knownvalue.StringExact("BOOLEAN"),
					"assignment_type": knownvalue.StringExact("NONE"),
				}),
				"summary": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":    knownvalue.StringExact("Summary of work"),
					"type":            knownvalue.StringExact("STRING"),
					"assignment_type": knownvalue.StringExact("SUMMARY"),
				}),
			})
	case "02_github_workflows":
		return xknownvalue.MapExact(map[string]knownvalue.Check{
				"workflow_ref": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":                   knownvalue.StringExact("Workflow Reference"),
					"type":                           knownvalue.StringExact("STRING"),
					"assignment_type":                knownvalue.StringExact("USER_INPUT"),
					"is_environment":                 knownvalue.Bool(false),
					"updateable_by_consumer":         knownvalue.Bool(false),
					"description":                    knownvalue.Null(),
					"selectable_values":              knownvalue.Null(),
					"value_validation_regex":         knownvalue.Null(),
					"validation_regex_error_message": knownvalue.Null(),
					"argument":                       knownvalue.Null(),
					"default_value":                  knownvalue.Null(),
					"sensitive":                      knownvalue.Null(),
				}),
			}),
			xknownvalue.MapExact(map[string]knownvalue.Check{
				"manual":                knownvalue.Null(),
				"github_workflows":      checkGithubWorkflowsImplementation(),
				"gitlab_pipeline":       knownvalue.Null(),
				"azure_devops_pipeline": knownvalue.Null(),
				"terraform":             knownvalue.Null(),
			}),
			xknownvalue.MapExact(map[string]knownvalue.Check{
				"workflow_run_url": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":    knownvalue.StringExact("Workflow Run URL"),
					"type":            knownvalue.StringExact("STRING"),
					"assignment_type": knownvalue.StringExact("RESOURCE_URL"),
				}),
			})
	case "03_manual":
		return xknownvalue.MapExact(map[string]knownvalue.Check{
				"approval_required": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":                   knownvalue.StringExact("Approval Required"),
					"type":                           knownvalue.StringExact("BOOLEAN"),
					"assignment_type":                knownvalue.StringExact("PLATFORM_OPERATOR_MANUAL_INPUT"),
					"is_environment":                 knownvalue.Bool(false),
					"updateable_by_consumer":         knownvalue.Bool(false),
					"description":                    knownvalue.Null(),
					"selectable_values":              knownvalue.Null(),
					"value_validation_regex":         knownvalue.Null(),
					"validation_regex_error_message": knownvalue.Null(),
					"argument":                       knownvalue.Null(),
					"default_value":                  knownvalue.Null(),
					"sensitive":                      knownvalue.Null(),
				}),
			}),
			xknownvalue.MapExact(map[string]knownvalue.Check{
				"manual":                checkManualImplementation(),
				"github_workflows":      knownvalue.Null(),
				"gitlab_pipeline":       knownvalue.Null(),
				"azure_devops_pipeline": knownvalue.Null(),
				"terraform":             knownvalue.Null(),
			}),
			xknownvalue.MapExact(map[string]knownvalue.Check{
				"approval_required": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":    knownvalue.StringExact("Approval Required"),
					"type":            knownvalue.StringExact("BOOLEAN"),
					"assignment_type": knownvalue.StringExact("NONE"),
				}),
			})
	case "04_azure_devops_pipeline":
		return xknownvalue.MapExact(map[string]knownvalue.Check{
				"pipeline_config": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":                   knownvalue.StringExact("Pipeline Configuration"),
					"type":                           knownvalue.StringExact("STRING"),
					"assignment_type":                knownvalue.StringExact("USER_INPUT"),
					"is_environment":                 knownvalue.Bool(false),
					"updateable_by_consumer":         knownvalue.Bool(false),
					"description":                    knownvalue.Null(),
					"selectable_values":              knownvalue.Null(),
					"value_validation_regex":         knownvalue.Null(),
					"validation_regex_error_message": knownvalue.Null(),
					"argument":                       knownvalue.Null(),
					"default_value":                  knownvalue.Null(),
					"sensitive":                      knownvalue.Null(),
				}),
			}),
			xknownvalue.MapExact(map[string]knownvalue.Check{
				"manual":                knownvalue.Null(),
				"github_workflows":      knownvalue.Null(),
				"gitlab_pipeline":       knownvalue.Null(),
				"azure_devops_pipeline": checkAzureDevopsPipelineImplementation(),
				"terraform":             knownvalue.Null(),
			}),
			xknownvalue.MapExact(map[string]knownvalue.Check{
				"pipeline_run_id": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":    knownvalue.StringExact("Pipeline Run ID"),
					"type":            knownvalue.StringExact("STRING"),
					"assignment_type": knownvalue.StringExact("NONE"),
				}),
			})
	case "05_gitlab_pipeline":
		return xknownvalue.MapExact(map[string]knownvalue.Check{
				"deployment_env": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":                   knownvalue.StringExact("Deployment Environment"),
					"type":                           knownvalue.StringExact("STRING"),
					"assignment_type":                knownvalue.StringExact("USER_INPUT"),
					"is_environment":                 knownvalue.Bool(false),
					"updateable_by_consumer":         knownvalue.Bool(false),
					"description":                    knownvalue.Null(),
					"selectable_values":              knownvalue.Null(),
					"value_validation_regex":         knownvalue.Null(),
					"validation_regex_error_message": knownvalue.Null(),
					"argument":                       knownvalue.Null(),
					"default_value":                  knownvalue.Null(),
					"sensitive":                      knownvalue.Null(),
				}),
			}),
			xknownvalue.MapExact(map[string]knownvalue.Check{
				"manual":                knownvalue.Null(),
				"github_workflows":      knownvalue.Null(),
				"gitlab_pipeline":       checkGitlabPipelineImplementation(),
				"azure_devops_pipeline": knownvalue.Null(),
				"terraform":             knownvalue.Null(),
			}),
			xknownvalue.MapExact(map[string]knownvalue.Check{
				"pipeline_web_url": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":    knownvalue.StringExact("Pipeline URL"),
					"type":            knownvalue.StringExact("STRING"),
					"assignment_type": knownvalue.StringExact("RESOURCE_URL"),
				}),
			})
	default:
		panic(fmt.Sprintf("unknown example suffix: %s", exampleSuffix))
	}
}

func checkTerraformImplementation() knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"terraform_version":              knownvalue.StringExact("1.9.0"),
		"repository_url":                 knownvalue.StringExact("https://github.com/example/building-block.git"),
		"async":                          knownvalue.Bool(true),
		"repository_path":                knownvalue.StringExact("terraform/modules/example"),
		"ref_name":                       knownvalue.StringExact("v1.0.0"),
		"use_mesh_http_backend_fallback": knownvalue.Bool(true),
		"ssh_known_host": xknownvalue.MapExact(map[string]knownvalue.Check{
			"host":      knownvalue.StringExact("github.com"),
			"key_type":  knownvalue.StringExact("ssh-rsa"),
			"key_value": xknownvalue.NotEmptyString(),
		}),
		"ssh_private_key": xknownvalue.MapExact(map[string]knownvalue.Check{
			"secret_value":   knownvalue.Null(),
			"secret_hash":    xknownvalue.NotEmptyString(),
			"secret_version": xknownvalue.NotEmptyString(),
		}),
		"pre_run_script": knownvalue.StringExact(`echo "hello world"`),
	})
}

func checkManualImplementation() knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{})
}

func checkGitlabPipelineImplementation() knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"project_id": knownvalue.StringExact("12345678"),
		"ref_name":   knownvalue.StringExact("main"),
		"pipeline_trigger_token": xknownvalue.MapExact(map[string]knownvalue.Check{
			"secret_value":   knownvalue.Null(),
			"secret_hash":    xknownvalue.NotEmptyString(),
			"secret_version": xknownvalue.NotEmptyString(),
		}),
		"integration_ref": xknownvalue.MapExact(map[string]knownvalue.Check{
			"uuid": xknownvalue.NotEmptyString(),
			"kind": knownvalue.StringExact("meshIntegration"),
		}),
	})
}

func checkGithubWorkflowsImplementation() knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"repository":            knownvalue.StringExact("example/building-block"),
		"branch":                knownvalue.StringExact("main"),
		"apply_workflow":        knownvalue.StringExact("apply.yml"),
		"destroy_workflow":      knownvalue.Null(),
		"async":                 knownvalue.Bool(true),
		"omit_run_object_input": knownvalue.Bool(true),
		"integration_ref": xknownvalue.MapExact(map[string]knownvalue.Check{
			"uuid": xknownvalue.NotEmptyString(),
			"kind": knownvalue.StringExact("meshIntegration"),
		}),
	})
}

func checkAzureDevopsPipelineImplementation() knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"project":     knownvalue.StringExact("MyProject"),
		"pipeline_id": knownvalue.StringExact("42"),
		"ref_name":    knownvalue.StringExact("refs/heads/main"),
		"async":       knownvalue.Bool(false),
		"integration_ref": xknownvalue.MapExact(map[string]knownvalue.Check{
			"uuid": xknownvalue.NotEmptyString(),
			"kind": knownvalue.StringExact("meshIntegration"),
		}),
	})
}

func TestAccBuildingBlockDefinitionSymbolValidation(t *testing.T) {
	// Symbol validation is client-side only; success cases need a real workspace in acceptance mode.
	if !IsMockClientTest() {
		t.Skip("symbol validation is tested with mock client only")
	}

	t.Parallel()

	// symbolConfig wraps a symbol value into a minimal valid BBD config.
	symbolConfig := func(symbol string) string {
		return fmt.Sprintf(`
resource "meshstack_building_block_definition" "test" {
  metadata = { owned_by_workspace = "my-workspace" }
  spec = {
    display_name = "Test"
    description  = "Test"
    symbol       = %q
  }
  version_spec = {
    draft = true
    implementation = { manual = {} }
  }
}`, symbol)
	}

	tests := []struct {
		name        string
		symbol      string
		expectError *regexp.Regexp
	}{
		{
			name:   "https URL",
			symbol: "https://example.com/icon.png",
		},
		{
			name:   "http URL",
			symbol: "http://example.com/icon.png",
		},
		{
			name:        "plain string is rejected",
			symbol:      "not-a-url-or-data-uri",
			expectError: regexp.MustCompile(`Invalid Symbol Format`),
		},
		{
			name:        "disallowed image type is rejected",
			symbol:      "data:image/bmp;base64," + base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 50))),
			expectError: regexp.MustCompile(`Invalid Symbol Format`),
		},
		{
			name:        "invalid base64 is rejected",
			symbol:      "data:image/png;base64,!!!not-valid-base64!!!",
			expectError: regexp.MustCompile(`Invalid Base64 in Symbol Data URI`),
		},
		{
			name:   "data URI decoded size exactly at 100 KiB limit",
			symbol: "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 100*1024))),
		},
		{
			name:        "data URI decoded size exceeds 100 KiB limit",
			symbol:      "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 100*1024+1))),
			expectError: regexp.MustCompile(`Symbol Image Too Large`),
		},
		{
			name:   "raw (no-padding) base64",
			symbol: "data:image/jpeg;base64," + base64.RawStdEncoding.EncodeToString([]byte(strings.Repeat("x", 100*1024))),
		},
		{
			name:   "svg+xml image type",
			symbol: "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(strings.Repeat("y", 50))),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := resource.TestStep{
				Config: symbolConfig(tt.symbol),
			}
			if tt.expectError != nil {
				step.ExpectError = tt.expectError
			}
			ApplyAndTest(t, resource.TestCase{
				Steps: []resource.TestStep{step},
			})
		})
	}
}
