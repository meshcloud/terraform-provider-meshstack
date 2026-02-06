package provider

import (
	_ "embed"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/stretchr/testify/assert"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/client/types/enum"
	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/clientmock"
)

func TestAccBuildingBlockDefinition(t *testing.T) {
	runBuildingBlockDefinitionTestCases(t)
}

func TestBuildingBlockDefinition(t *testing.T) {
	// Run acceptance tests as unit tests with mock
	runBuildingBlockDefinitionTestCases(t, SetupMockClient(func(t *testing.T, testCase *resource.TestCase, mockClient clientmock.Client) {
		t.Helper()
		// Determine the test case name from test name
		testName := t.Name()
		is01Terraform := strings.Contains(testName, "01_terraform")

		// 01_terraform creates an extra dependency BBD, so it has 2 BBDs total
		expectedBBDCount := 1
		if is01Terraform {
			expectedBBDCount = 2
		}

		for stepIdx := range testCase.Steps {
			if stepIdx == 3 {
				// Step 4 changes draft=false->true and creates new draft version
				testCase.Steps[stepIdx].PostApplyFunc = func() {
					assert.Len(t, mockClient.BuildingBlockDefinition.Store, expectedBBDCount)
					assert.Len(t, mockClient.BuildingBlockDefinitionVersion.Store, 2*expectedBBDCount) // 2 versions per BBD
				}
			} else {
				testCase.Steps[stepIdx].PostApplyFunc = func() {
					assert.Len(t, mockClient.BuildingBlockDefinition.Store, expectedBBDCount)
					assert.Len(t, mockClient.BuildingBlockDefinitionVersion.Store, expectedBBDCount)
				}
			}
		}
	}))
}

func runBuildingBlockDefinitionTestCases(t *testing.T, testCaseModifiers ...ResourceTestCaseModifier) {
	t.Helper()

	var (
		versionStateDraft    = client.MeshBuildingBlockDefinitionVersionStateDraft
		versionStateReleased = client.MeshBuildingBlockDefinitionVersionStateReleased
	)

	expectedVersion := func(number int64, state enum.Entry[client.MeshBuildingBlockDefinitionVersionState]) knownvalue.Check {
		return knownvalue.MapExact(map[string]knownvalue.Check{
			"uuid":         KnownValueNotEmptyString(),
			"number":       knownvalue.Int64Exact(number),
			"state":        knownvalue.StringExact(state.String()),
			"content_hash": KnownValueNotEmptyString(),
		})
	}

	for exampleResource := range (examples.Resource{Name: "building_block_definition"}).All() {
		exampleSuffix := strings.TrimPrefix(exampleResource.Suffix, "_")
		t.Run(exampleSuffix, func(t *testing.T) {
			t.Parallel()
			var resourceAddress examples.Identifier
			config := exampleResource.Config().
				SingleResourceAddress(&resourceAddress).
				OwnedByAdminWorkspace()

			if exampleSuffix == "01_terraform" {
				// Terraform example is a bit more involved as it has dependencies
				var environmentTagAddress, costCenterTagAddress, dependencyBBDAddress examples.Identifier
				// ensure tag definitions, platform, and dependency BBD are created before resource under test
				config = config.
					Join(
						exampleResource.TestSupportConfig("_shared"),
						exampleResource.TestSupportConfig("_tag-environment").
							SingleResourceAddress(&environmentTagAddress),
						exampleResource.TestSupportConfig("_tag-cost-center").
							SingleResourceAddress(&costCenterTagAddress),
						exampleResource.TestSupportConfig("_dependency-bbd").
							SingleResourceAddress(&dependencyBBDAddress).
							OwnedByAdminWorkspace(),
					).
					ReplaceAll(`"environment" = [`, environmentTagAddress.Format(`(%s.spec.key) = [`)).
					ReplaceAll(`"cost-center" = [`, costCenterTagAddress.Format(`(%s.spec.key) = [`)).
					ReplaceAll(`dependency_refs = [{ uuid = "d161e3bf-c3e7-45f2-aa21-28de14593a74" }]`, dependencyBBDAddress.Format(`dependency_refs = [%s.ref]`)).
					ReplaceAll(`notification_subscribers  = ["user:some-username", "email:ops@example.com"]`, `notification_subscribers = ["email:ops@example.com"]`)
			}
			const bbdDescription = "An example building block definition"

			testSteps := []resource.TestStep{
				// Step 1: Create resource and validate state thoroughly!
				{
					Config: config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("metadata"), checkBuildingBlockMetadata(exampleSuffix != "01_terraform")),
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("spec"), checkBuildingBlockSpec(bbdDescription, exampleSuffix != "01_terraform")),
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec(exampleSuffix, versionStateDraft, 1)),

						// Version checks - only one draft version exists, so version_latest_release is not set
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("version_latest_release"), knownvalue.Null()),
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{expectedVersion(1, versionStateDraft)})),
					},
				},
			}

			if exampleSuffix == "03_manual" {
				configSpecChange := config.ReplaceAll(bbdDescription, "An updated building block definition")
				configDraftFalse := config.ReplaceAll("draft = true", "draft = false")
				configDraftTrueAgain := configSpecChange

				testSteps = append(testSteps,
					// Step 2: Update BBD Spec, which will not trigger a new BBD version
					resource.TestStep{
						Config: configSpecChange.String(),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{
								plancheck.ExpectResourceAction(resourceAddress.String(), plancheck.ResourceActionUpdate),
							},
						},
						ConfigStateChecks: []statecheck.StateCheck{
							statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("metadata"), checkBuildingBlockMetadata(exampleSuffix != "01_terraform")),
							statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("spec"), checkBuildingBlockSpec("An updated building block definition", exampleSuffix != "01_terraform")),
							// Version checks - nothing has changed
							statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{expectedVersion(1, versionStateDraft)})),
						},
					},
					// Step 3: Update BBD Version Spec with draft=false, which will release the existing BBD version
					resource.TestStep{
						Config: configDraftFalse.String(),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{
								plancheck.ExpectResourceAction(resourceAddress.String(), plancheck.ResourceActionUpdate),
							},
						},
						ConfigStateChecks: []statecheck.StateCheck{
							statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec(exampleSuffix, versionStateReleased, 1)),
							// Version checks - draft is now released, so 'version_latest_release' becomes set (content hash does not change though)
							statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
							statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("version_latest"), expectedVersion(1, versionStateReleased)),
							statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{expectedVersion(1, versionStateReleased)})),
						},
					},
					// Step 4: Update BBD Version Spec with draft=true again, which will create a new draft version
					resource.TestStep{
						Config: configDraftTrueAgain.String(),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{
								plancheck.ExpectResourceAction(resourceAddress.String(), plancheck.ResourceActionUpdate),
							},
						},
						ConfigStateChecks: []statecheck.StateCheck{
							statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec(exampleSuffix, versionStateDraft, 2)),
							// Version checks - a new draft is added
							statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
							statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("version_latest"), expectedVersion(2, versionStateDraft)),
							statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{
								expectedVersion(1, versionStateReleased),
								expectedVersion(2, versionStateDraft),
							})),
						},
					},
				)
			}
			ResourceTestCaseModifiers(testCaseModifiers).
				ApplyAndTest(t, resource.TestCase{Steps: testSteps})
		})
	}
}

func checkBuildingBlockMetadata(optional bool) knownvalue.Check {
	expected := map[string]knownvalue.Check{
		"uuid":               KnownValueNotEmptyString(),
		"owned_by_workspace": knownvalue.StringExact("managed-customer"),
		"tags":               knownvalue.MapSizeExact(2),
	}
	if optional {
		expected["tags"] = knownvalue.Null()
	}
	return knownvalue.MapExact(expected)
}

func checkBuildingBlockSpec(expectedDescription string, optional bool) knownvalue.Check {
	expected := map[string]knownvalue.Check{
		"display_name":      knownvalue.StringExact("Example Building Block"),
		"symbol":            knownvalue.StringExact("🏗️"),
		"description":       knownvalue.StringExact(expectedDescription),
		"readme":            KnownValueNotEmptyString(),
		"support_url":       knownvalue.StringExact("https://support.example.com/building-blocks"),
		"documentation_url": knownvalue.StringExact("https://docs.example.com/building-blocks"),
		"target_type":       knownvalue.StringExact("TENANT_LEVEL"),
		"supported_platforms": knownvalue.ListExact([]knownvalue.Check{
			knownvalue.MapExact(map[string]knownvalue.Check{
				"kind": knownvalue.StringExact("meshPlatformType"),
				"name": knownvalue.StringExact("AZURE"), // Can be any platform in test
			}),
			knownvalue.MapExact(map[string]knownvalue.Check{
				"kind": knownvalue.StringExact("meshPlatformType"),
				"name": knownvalue.StringExact("AWS"), // Can be any platform in test
			}),
		}),
		"run_transparency":          knownvalue.Bool(true),
		"use_in_landing_zones_only": knownvalue.Bool(true),
		"notification_subscribers": knownvalue.ListExact([]knownvalue.Check{
			knownvalue.StringExact("email:ops@example.com"),
		}),
	}
	if optional {
		for _, nullKey := range []string{"readme", "support_url", "documentation_url", "supported_platforms", "notification_subscribers"} {
			expected[nullKey] = knownvalue.Null()
		}
		expected["target_type"] = knownvalue.StringExact("WORKSPACE_LEVEL")
		expected["run_transparency"] = knownvalue.Bool(false)
		expected["use_in_landing_zones_only"] = knownvalue.Bool(false)
		expected["symbol"] = KnownValueNotEmptyString()
	}
	return knownvalue.MapExact(expected)
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
	expected := map[string]knownvalue.Check{
		"state":                      knownvalue.StringExact(expectedState.String()),
		"version_number":             knownvalue.Int64Exact(expectedNumber),
		"draft":                      knownvalue.Bool(expectedState == client.MeshBuildingBlockDefinitionVersionStateDraft),
		"only_apply_once_per_tenant": knownvalue.Bool(false),
		"deletion_mode":              knownvalue.StringExact("DELETE"),
		"runner_ref": knownvalue.MapExact(map[string]knownvalue.Check{
			"kind": knownvalue.StringExact("meshBuildingBlockRunner"),
			"uuid": knownvalue.StringExact(SharedBuildingBlockRunnerUuids[exampleSuffixesToImplementationType[exampleSuffix]]),
		}),
		"dependency_refs": knownvalue.Null(),
		"inputs":          checkInputs,
		"implementation":  checkImplementation,
		"outputs":         checkOutputs,
	}

	if exampleSuffix == "01_terraform" {
		expected["dependency_refs"] = knownvalue.ListExact([]knownvalue.Check{
			knownvalue.MapExact(map[string]knownvalue.Check{
				"kind": knownvalue.StringExact("meshBuildingBlockDefinition"),
				"uuid": KnownValueNotEmptyString(),
			}),
		})
	}
	return knownvalue.MapExact(expected)
}

func checksForImplementation(exampleSuffix string) (checkInputs, checkImplementation, checkOutputs knownvalue.Check) {
	switch exampleSuffix {
	case "01_terraform":
		return knownvalue.MapExact(map[string]knownvalue.Check{
				"environment": knownvalue.MapExact(map[string]knownvalue.Check{
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
				"resource_name": knownvalue.MapExact(map[string]knownvalue.Check{
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
				"SOMETHING_VERY_SECRET": knownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":           knownvalue.StringExact("Top Secret"),
					"type":                   knownvalue.StringExact("STRING"),
					"assignment_type":        knownvalue.StringExact("STATIC"),
					"is_environment":         knownvalue.Bool(true),
					"updateable_by_consumer": knownvalue.Bool(false),
					"description":            knownvalue.StringExact("Really secret"),
					"sensitive": knownvalue.MapExact(map[string]knownvalue.Check{
						"argument": knownvalue.MapExact(map[string]knownvalue.Check{
							"value":       knownvalue.Null(),
							"hash":        KnownValueNotEmptyString(),
							"fingerprint": KnownValueNotEmptyString(),
						}),
						"default_value": knownvalue.Null(),
					}),
					"value_validation_regex":         knownvalue.Null(),
					"validation_regex_error_message": knownvalue.Null(),
					"selectable_values":              knownvalue.Null(),
					"argument":                       knownvalue.Null(),
					"default_value":                  knownvalue.Null(),
				}),
			}),
			knownvalue.MapExact(map[string]knownvalue.Check{
				"manual":                knownvalue.Null(),
				"github_workflows":      knownvalue.Null(),
				"gitlab_pipeline":       knownvalue.Null(),
				"azure_devops_pipeline": knownvalue.Null(),
				"terraform":             checkTerraformImplementation(),
			}), knownvalue.MapExact(map[string]knownvalue.Check{
				"some_output_flag": knownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":    knownvalue.StringExact("If true, it really worked"),
					"type":            knownvalue.StringExact("BOOLEAN"),
					"assignment_type": knownvalue.StringExact("NONE"),
				}),
				"summary": knownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":    knownvalue.StringExact("Summary of work"),
					"type":            knownvalue.StringExact("STRING"),
					"assignment_type": knownvalue.StringExact("SUMMARY"),
				}),
			})
	case "02_github_workflows":
		return knownvalue.MapExact(map[string]knownvalue.Check{
				"workflow_ref": knownvalue.MapExact(map[string]knownvalue.Check{
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
			knownvalue.MapExact(map[string]knownvalue.Check{
				"manual":                knownvalue.Null(),
				"github_workflows":      checkGithubWorkflowsImplementation(),
				"gitlab_pipeline":       knownvalue.Null(),
				"azure_devops_pipeline": knownvalue.Null(),
				"terraform":             knownvalue.Null(),
			}),
			knownvalue.MapExact(map[string]knownvalue.Check{
				"workflow_run_url": knownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":    knownvalue.StringExact("Workflow Run URL"),
					"type":            knownvalue.StringExact("STRING"),
					"assignment_type": knownvalue.StringExact("RESOURCE_URL"),
				}),
			})
	case "03_manual":
		return knownvalue.MapExact(map[string]knownvalue.Check{
				"approval_required": knownvalue.MapExact(map[string]knownvalue.Check{
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
			knownvalue.MapExact(map[string]knownvalue.Check{
				"manual":                checkManualImplementation(),
				"github_workflows":      knownvalue.Null(),
				"gitlab_pipeline":       knownvalue.Null(),
				"azure_devops_pipeline": knownvalue.Null(),
				"terraform":             knownvalue.Null(),
			}),
			knownvalue.MapExact(map[string]knownvalue.Check{
				"completion_status": knownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":    knownvalue.StringExact("Completion Status"),
					"type":            knownvalue.StringExact("STRING"),
					"assignment_type": knownvalue.StringExact("SUMMARY"),
				}),
			})
	case "04_azure_devops_pipeline":
		return knownvalue.MapExact(map[string]knownvalue.Check{
				"pipeline_config": knownvalue.MapExact(map[string]knownvalue.Check{
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
			knownvalue.MapExact(map[string]knownvalue.Check{
				"manual":                knownvalue.Null(),
				"github_workflows":      knownvalue.Null(),
				"gitlab_pipeline":       knownvalue.Null(),
				"azure_devops_pipeline": checkAzureDevopsPipelineImplementation(),
				"terraform":             knownvalue.Null(),
			}),
			knownvalue.MapExact(map[string]knownvalue.Check{
				"pipeline_run_id": knownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":    knownvalue.StringExact("Pipeline Run ID"),
					"type":            knownvalue.StringExact("STRING"),
					"assignment_type": knownvalue.StringExact("NONE"),
				}),
			})
	case "05_gitlab_pipeline":
		return knownvalue.MapExact(map[string]knownvalue.Check{
				"deployment_env": knownvalue.MapExact(map[string]knownvalue.Check{
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
			knownvalue.MapExact(map[string]knownvalue.Check{
				"manual":                knownvalue.Null(),
				"github_workflows":      knownvalue.Null(),
				"gitlab_pipeline":       checkGitlabPipelineImplementation(),
				"azure_devops_pipeline": knownvalue.Null(),
				"terraform":             knownvalue.Null(),
			}),
			knownvalue.MapExact(map[string]knownvalue.Check{
				"pipeline_web_url": knownvalue.MapExact(map[string]knownvalue.Check{
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
	return knownvalue.MapExact(map[string]knownvalue.Check{
		"terraform_version":              knownvalue.StringExact("1.9.0"),
		"repository_url":                 knownvalue.StringExact("https://github.com/example/building-block.git"),
		"async":                          knownvalue.Bool(false),
		"repository_path":                knownvalue.StringExact("terraform/modules/example"),
		"ref_name":                       knownvalue.StringExact("v1.0.0"),
		"use_mesh_http_backend_fallback": knownvalue.Bool(false),
		"ssh_known_host": knownvalue.MapExact(map[string]knownvalue.Check{
			"host":      knownvalue.StringExact("github.com"),
			"key_type":  knownvalue.StringExact("ssh-rsa"),
			"key_value": KnownValueNotEmptyString(),
		}),
		"ssh_private_key": knownvalue.MapExact(map[string]knownvalue.Check{
			"value":       knownvalue.Null(),
			"hash":        KnownValueNotEmptyString(),
			"fingerprint": KnownValueNotEmptyString(),
		}),
	})
}

func checkManualImplementation() knownvalue.Check {
	return knownvalue.MapExact(map[string]knownvalue.Check{})
}

func checkGitlabPipelineImplementation() knownvalue.Check {
	return knownvalue.MapExact(map[string]knownvalue.Check{
		"project_id": knownvalue.StringExact("12345678"),
		"ref_name":   knownvalue.StringExact("main"),
		"pipeline_trigger_token": knownvalue.MapExact(map[string]knownvalue.Check{
			"value":       knownvalue.Null(),
			"hash":        knownvalue.StringExact("sha256:glptt-..."),
			"fingerprint": knownvalue.StringExact("sha256:glptt-..."),
		}),
		"integration_ref": knownvalue.MapExact(map[string]knownvalue.Check{
			"uuid": knownvalue.StringExact("550e8400-e29b-41d4-a716-446655440000"),
			"kind": knownvalue.StringExact("meshIntegration"),
		}),
	})
}

func checkGithubWorkflowsImplementation() knownvalue.Check {
	return knownvalue.MapExact(map[string]knownvalue.Check{
		"repository":            knownvalue.StringExact("example/building-block"),
		"branch":                knownvalue.StringExact("main"),
		"apply_workflow":        knownvalue.StringExact("apply.yml"),
		"destroy_workflow":      knownvalue.Null(),
		"async":                 knownvalue.Bool(false),
		"omit_run_object_input": knownvalue.Bool(false),
		"integration_ref": knownvalue.MapExact(map[string]knownvalue.Check{
			"uuid": knownvalue.StringExact("550e8400-e29b-41d4-a716-446655440000"),
			"kind": knownvalue.StringExact("meshIntegration"),
		}),
	})
}

func checkAzureDevopsPipelineImplementation() knownvalue.Check {
	return knownvalue.MapExact(map[string]knownvalue.Check{
		"project":     knownvalue.StringExact("MyProject"),
		"pipeline_id": knownvalue.StringExact("42"),
		"async":       knownvalue.Bool(false),
		"integration_ref": knownvalue.MapExact(map[string]knownvalue.Check{
			"uuid": knownvalue.StringExact("550e8400-e29b-41d4-a716-446655440000"),
			"kind": knownvalue.StringExact("meshIntegration"),
		}),
	})
}
