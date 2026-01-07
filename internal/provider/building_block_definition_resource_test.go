package provider

import (
	_ "embed"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/meshcloud/terraform-provider-meshstack/examples"
)

func TestAccBuildingBlockDefinition(t *testing.T) {
	const resourceAddress = "meshstack_building_block_definition.example"
	const resourceIdentifier = "my-building-block-def"
	const workspaceIdentifier = "my-workspace"

	// Define expected version objects
	version1 := knownvalue.MapExact(map[string]knownvalue.Check{
		"uuid":   knownvalue.StringExact("dummy-version-uuid-1"),
		"number": knownvalue.Int64Exact(1),
		"state":  knownvalue.StringExact("RELEASED"),
	})

	version2 := knownvalue.MapExact(map[string]knownvalue.Check{
		"uuid":   knownvalue.StringExact("dummy-version-uuid-2"),
		"number": knownvalue.Int64Exact(2),
		"state":  knownvalue.StringExact("DRAFT"),
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		IsUnitTest:               true,
		Steps: []resource.TestStep{
			{
				Config: examples.Resource{Name: "building_block_definition"}.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					// Metadata checks
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata"), knownvalue.MapExact(map[string]knownvalue.Check{
						"uuid":                   knownvalue.StringExact("dummy-uuid-12345"),
						"owned_by_workspace":     knownvalue.StringExact(workspaceIdentifier),
						"created_on":             knownvalue.NotNull(),
						"marked_for_deletion_on": knownvalue.Null(),
						"marked_for_deletion_by": knownvalue.Null(),
						"tags": knownvalue.MapExact(map[string]knownvalue.Check{
							"environment": knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("production"), knownvalue.StringExact("staging")}),
							"team":        knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("platform-team")}),
							"cost-center": knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("cc-123")}),
						}),
					})),

					// Spec checks
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec"), knownvalue.MapExact(map[string]knownvalue.Check{
						"display_name":                      knownvalue.StringExact("Example Building Block"),
						"symbol":                            knownvalue.StringExact("🏗️"),
						"description":                       knownvalue.StringExact("An example building block definition"),
						"readme":                            knownvalue.StringExact("# Example Building Block\n\nThis is a comprehensive example showcasing all available attributes."),
						"support_url":                       knownvalue.StringExact("https://support.example.com/building-blocks"),
						"documentation_url":                 knownvalue.StringExact("https://docs.example.com/building-blocks"),
						"target_type":                       knownvalue.StringExact(TenantTargetType),
						"supported_platforms":               knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("azure.platform"), knownvalue.StringExact("aws.platform")}),
						"use_in_landing_zones_only":         knownvalue.Bool(true),
						"run_transparency":                  knownvalue.Bool(true),
						"notification_subscriber_usernames": knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("admin@example.com"), knownvalue.StringExact("ops@example.com")}),
					})),

					// Draft and runner ref
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("draft"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("runner_ref"), knownvalue.StringExact("my-runner")),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("only_apply_once_per_tenant"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("deletion_mode"), knownvalue.StringExact("DELETE")),

					// Dependency refs
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("dependency_refs"), knownvalue.ListExact([]knownvalue.Check{
						knownvalue.StringExact("dep-1"),
						knownvalue.StringExact("dep-2"),
					})),

					// Inputs check
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("inputs"), knownvalue.MapExact(map[string]knownvalue.Check{
						"environment": knownvalue.MapExact(map[string]knownvalue.Check{
							"display_name":                   knownvalue.StringExact("Environment"),
							"type":                           knownvalue.StringExact("SINGLE_SELECT"),
							"assignment_type":                knownvalue.StringExact("USER_INPUT"),
							"argument":                       knownvalue.Null(),
							"is_environment":                 knownvalue.Bool(false),
							"is_sensitive":                   knownvalue.Bool(false),
							"updateable_by_consumer":         knownvalue.Bool(true),
							"selectable_values":              knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("dev"), knownvalue.StringExact("staging"), knownvalue.StringExact("prod")}),
							"default_value":                  knownvalue.Null(),
							"description":                    knownvalue.StringExact("The target environment"),
							"value_validation_regex":         knownvalue.Null(),
							"validation_regex_error_message": knownvalue.Null(),
						}),
						"resource_name": knownvalue.MapExact(map[string]knownvalue.Check{
							"display_name":                   knownvalue.StringExact("Resource Name"),
							"type":                           knownvalue.StringExact("STRING"),
							"assignment_type":                knownvalue.StringExact("USER_INPUT"),
							"argument":                       knownvalue.Null(),
							"is_environment":                 knownvalue.Bool(false),
							"is_sensitive":                   knownvalue.Bool(false),
							"updateable_by_consumer":         knownvalue.Bool(true),
							"selectable_values":              knownvalue.Null(),
							"default_value":                  knownvalue.Null(),
							"description":                    knownvalue.StringExact("Name of the resource to create"),
							"value_validation_regex":         knownvalue.StringExact("^[a-z0-9-]+$"),
							"validation_regex_error_message": knownvalue.StringExact("Resource name must contain only lowercase letters, numbers, and hyphens"),
						}),
					})),

					// Outputs check
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("outputs"), knownvalue.MapExact(map[string]knownvalue.Check{
						"tenant_id": knownvalue.MapExact(map[string]knownvalue.Check{
							"display_name":    knownvalue.StringExact("Tenant ID"),
							"type":            knownvalue.StringExact("STRING"),
							"assignment_type": knownvalue.StringExact("PLATFORM_TENANT_ID"),
						}),
						"sign_in_url": knownvalue.MapExact(map[string]knownvalue.Check{
							"display_name":    knownvalue.StringExact("Sign-in URL"),
							"type":            knownvalue.StringExact("STRING"),
							"assignment_type": knownvalue.StringExact("SIGN_IN_URL"),
						}),
					})),

					// Implementation check (terraform block)
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("implementation"), knownvalue.MapExact(map[string]knownvalue.Check{
						"terraform": knownvalue.MapExact(map[string]knownvalue.Check{
							"terraform_version":              knownvalue.StringExact("1.9.0"),
							"repository_url":                 knownvalue.StringExact("https://github.com/example/building-block.git"),
							"async":                          knownvalue.Bool(false),
							"repository_path":                knownvalue.StringExact("terraform/modules/example"),
							"ref_name":                       knownvalue.StringExact("v1.0.0"),
							"use_mesh_http_backend_fallback": knownvalue.Bool(false),
							"ssh_private_key":                knownvalue.Null(), // write-only, should NOT be in state
							"ssh_private_key_version":        knownvalue.StringExact("v1"),
							"ssh_known_host": knownvalue.MapExact(map[string]knownvalue.Check{
								"host":      knownvalue.StringExact("github.com"),
								"key_type":  knownvalue.StringExact("ssh-rsa"),
								"key_value": knownvalue.StringExact("AAAAB3NzaC1yc2EAAAABIwAAAQEAq2A7hRGmdnm9tUDbO9IDSwBK6TbQa+..."),
							}),
						}),
						"github_actions": knownvalue.Null(), // not used in this example
					})),

					// Version checks
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("version_latest"), version2),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("version_latest_release"), version1),
					statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{
						version1,
						version2,
					})),
				},
			},
		},
	})
}
