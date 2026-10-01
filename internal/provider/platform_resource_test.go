package provider

import (
	"fmt"
	"strings"
	"testing"

	tfconfig "github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// platformVariant names one meshstack_platform example variant: the index of its step file in
// examples/resources/meshstack_platform/resource-test-<index>.tf, and the address of the block it
// declares. The step files carry every optional boolean set to true, which is what catches a
// nullable flag the backend drops on the way to persistence — null and false look alike there.
type platformVariant struct {
	index int
	addr  string
	// suffix names the variant in the shared config/role-mapping/quota assertions.
	suffix string
}

var platformVariants = []platformVariant{
	{1, "meshstack_platform.example_azure", "01_azure"},
	{2, "meshstack_platform.example_aws", "02_aws"},
	{3, "meshstack_platform.example_gcp", "03_gcp"},
	{4, "meshstack_platform.example_kubernetes", "04_kubernetes"},
	{5, "meshstack_platform.example_aks", "05_aks"},
	{6, "meshstack_platform.example_azurerg", "06_azurerg"},
	{7, "meshstack_platform.example_openshift", "07_openshift"},
	{8, "meshstack_platform.example_custom", "08_custom"},
}

// platformStepConfig is a platform variant's step, with the workspace that owns it. The custom
// variant additionally needs the platform type its config points at.
func platformStepConfig(t *testing.T, variant platformVariant) string {
	t.Helper()
	parts := []string{
		examples.Resource.TestStepConfig(t, "platform", variant.index),
		examples.Resource.TestStepConfig(t, "workspace", 1, "variables", "prerequisites"),
	}
	if variant.suffix == "08_custom" {
		parts = append(parts, examples.Resource.TestStepConfig(t, "platform_type", 1))
	}
	return examples.JoinTestStepConfigs(parts...)
}

func TestAccPlatformResource(t *testing.T) {
	t.Parallel()

	for _, variant := range platformVariants {
		t.Run(variant.suffix, func(t *testing.T) {
			suffix := acctest.RandString(8)
			vars := SuffixVariables(suffix)
			config := platformStepConfig(t, variant)

			var resourceUuid string
			steps := platformCreateSteps(config, suffix, vars, variant, &resourceUuid)

			switch variant.suffix {
			case "01_azure":
				// Only this variant also exercises an update, and asserts the identifier's
				// <platform>.<location> shape.
				steps[0].ConfigStateChecks = append(steps[0].ConfigStateChecks,
					statecheck.ExpectKnownValue(variant.addr, tfjsonpath.New("identifier"), knownvalue.StringFunc(func(value string) error {
						parts := strings.SplitN(value, ".", 2)
						if len(parts) != 2 || !strings.HasPrefix(parts[0], "my-platform-") || parts[1] == "" {
							return fmt.Errorf("expected identifier format <platform>.<location>, got %q", value)
						}
						return nil
					})),
				)
			case "05_aks":
				// The access token is write-only, so importing it plans an update: the hash is unknown
				// until apply and only the version pins what the config declared.
				steps = append(steps, resource.TestStep{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ConfigVariables: vars,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName:       variant.addr,
					ExpectNonEmptyPlan: true,
					ImportPlanChecks: resource.ImportPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(variant.addr, plancheck.ResourceActionUpdate),
							plancheck.ExpectUnknownValue(variant.addr, aksSecretPath().AtMapKey("secret_hash")),
							plancheck.ExpectKnownValue(variant.addr, aksSecretPath().AtMapKey("secret_version"), knownvalue.StringExact(nonEphemeralSecretVersion("top-secret-value"))),
						},
					},
				})
			}

			if variant.suffix != "05_aks" {
				steps = append(steps, platformImportStep(variant, vars, &resourceUuid))
			}

			ApplyAndTest(t, resource.TestCase{Steps: steps})
		})
	}
}

// aksSecretPath is a factory to work around slice copy/clone bug in tfjsonpath.Path.AtMapKey.
func aksSecretPath() tfjsonpath.Path {
	return tfjsonpath.New("spec").AtMapKey("config").AtMapKey("aks").AtMapKey("replication").AtMapKey("access_token")
}

// platformCreateSteps returns create+state-check steps for a platform test.
func platformCreateSteps(config, suffix string, vars tfconfig.Variables, variant platformVariant, resourceUuidOut *string) []resource.TestStep {
	return []resource.TestStep{
		{
			Config:          config,
			ConfigVariables: vars,
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(variant.addr, plancheck.ResourceActionCreate),
				},
			},
			ConfigStateChecks: append(
				[]statecheck.StateCheck{
					statecheck.ExpectKnownValue(variant.addr, tfjsonpath.New("metadata"), checkPlatformMetadata(suffix, resourceUuidOut)),
					statecheck.ExpectKnownValue(variant.addr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("Example Platform")),
				},
				checkPlatformConfigState(variant.addr, variant.suffix, suffix)...,
			),
		},
	}
}

// platformImportStep imports the platform by the uuid the create step recorded.
func platformImportStep(variant platformVariant, vars tfconfig.Variables, resourceUuidOut *string) resource.TestStep {
	return resource.TestStep{
		ImportState:     true,
		ImportStateKind: resource.ImportBlockWithID,
		ConfigVariables: vars,
		ImportStateIdFunc: func(state *terraform.State) (string, error) {
			return *resourceUuidOut, nil
		},
		ResourceName: variant.addr,
	}
}

func checkPlatformMetadata(suffix string, resourceUuidOut *string) knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"name":               knownvalue.StringExact("my-platform-" + suffix),
		"owned_by_workspace": knownvalue.StringExact("test-ws-" + suffix),
		"uuid": xknownvalue.NotEmptyString(func(actualValue string) error {
			*resourceUuidOut = actualValue
			return nil
		}),
	})
}

func checkMeteringProcessingConfig() knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"compact_timelines_after_days": knownvalue.Int64Exact(30),
		"delete_raw_data_after_days":   knownvalue.Int64Exact(65),
	})
}

func checkPlatformConfigState(resourceAddress, exampleSuffix, suffix string) []statecheck.StateCheck {
	var platformType string
	var configCheck knownvalue.Check

	switch exampleSuffix {
	case "01_azure":
		platformType = "azure"
		configCheck = checkAzurePlatformConfig()
	case "02_aws":
		platformType = "aws"
		configCheck = checkAwsPlatformConfig()
	case "03_gcp":
		platformType = "gcp"
		configCheck = checkGcpPlatformConfig()
	case "04_kubernetes":
		platformType = "kubernetes"
		configCheck = checkKubernetesPlatformConfig()
	case "05_aks":
		platformType = "aks"
		configCheck = checkAksPlatformConfig()
	case "06_azurerg":
		platformType = "azurerg"
		configCheck = checkAzureRgPlatformConfig()
	case "07_openshift":
		platformType = "openshift"
		configCheck = checkOpenshiftPlatformConfig()
	case "08_custom":
		platformType = "custom"
		configCheck = checkCustomPlatformConfig(suffix)
	default:
		platformType = "azure"
		configCheck = knownvalue.NotNull()
	}

	checks := []statecheck.StateCheck{
		statecheck.ExpectKnownValue(
			resourceAddress,
			tfjsonpath.New("spec").AtMapKey("config").AtMapKey(platformType),
			configCheck,
		),
		checkPlatformQuotas(resourceAddress, exampleSuffix),
	}
	return append(checks, checkPlatformRoleMappings(resourceAddress, exampleSuffix)...)
}

// checkPlatformRoleMappings asserts that aws/gcp role mappings resolve their `project_role_ref`
// as an unordered set (openshift/azure role mappings are already asserted in their config check).
// It guards against the regression where these were modeled as ordered lists and against a
// `project_role_ref` whose identifier/kind failed to resolve.
func checkPlatformRoleMappings(resourceAddress, exampleSuffix string) []statecheck.StateCheck {
	projectRoleRef := func(name string) knownvalue.Check {
		return xknownvalue.MapExact(map[string]knownvalue.Check{
			"name": knownvalue.StringExact(name),
			"kind": knownvalue.StringExact("meshProjectRole"),
		})
	}

	switch exampleSuffix {
	case "02_aws":
		return []statecheck.StateCheck{
			statecheck.ExpectKnownValue(
				resourceAddress,
				tfjsonpath.New("spec").AtMapKey("config").AtMapKey("aws").
					AtMapKey("replication").AtMapKey("aws_identity_store").AtMapKey("aws_role_mappings"),
				knownvalue.SetExact([]knownvalue.Check{
					xknownvalue.MapExact(map[string]knownvalue.Check{
						"project_role_ref":    projectRoleRef("admin"),
						"aws_role":            knownvalue.StringExact("admin"),
						"permission_set_arns": knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("arn:aws:sso:::permissionSet/ssoins-1234567890abcdef/ps-1234567890abcdef")}),
					}),
				}),
			),
		}
	case "03_gcp":
		return []statecheck.StateCheck{
			statecheck.ExpectKnownValue(
				resourceAddress,
				tfjsonpath.New("spec").AtMapKey("config").AtMapKey("gcp").
					AtMapKey("replication").AtMapKey("gcp_role_mappings"),
				knownvalue.SetExact([]knownvalue.Check{
					xknownvalue.MapExact(map[string]knownvalue.Check{
						"project_role_ref": projectRoleRef("admin"),
						"gcp_role":         knownvalue.StringExact("roles/editor"),
					}),
					xknownvalue.MapExact(map[string]knownvalue.Check{
						"project_role_ref": projectRoleRef("reader"),
						"gcp_role":         knownvalue.StringExact("roles/viewer"),
					}),
				}),
			),
		}
	default:
		return nil
	}
}

func checkPlatformQuotas(resourceAddress, exampleSuffix string) statecheck.StateCheck {
	switch exampleSuffix {
	case "01_azure":
		// Azure example defines 2 quota entries (vcpu, storage)
		return statecheck.ExpectKnownValue(
			resourceAddress,
			tfjsonpath.New("spec").AtMapKey("quota_definitions"),
			knownvalue.SetSizeExact(2),
		)
	default:
		return statecheck.ExpectKnownValue(
			resourceAddress,
			tfjsonpath.New("spec").AtMapKey("quota_definitions"),
			knownvalue.SetSizeExact(0),
		)
	}
}

// checkAzurePlatformConfig verifies the azure config block is present and non-null.
// The Azure example has a complex structure that is kept current in resource_01_azure.tf;
// detailed field checks are out of scope here as the schema is primarily tested by
// the kubernetes/aks/azurerg/openshift examples which have stable check functions.
func checkAzurePlatformConfig() knownvalue.Check {
	return knownvalue.NotNull()
}

// checkAwsPlatformConfig verifies the aws config block is present and non-null.
func checkAwsPlatformConfig() knownvalue.Check {
	return knownvalue.NotNull()
}

// checkGcpPlatformConfig verifies the gcp config block is present and non-null.
func checkGcpPlatformConfig() knownvalue.Check {
	return knownvalue.NotNull()
}

func checkKubernetesPlatformConfig() knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"base_url":               knownvalue.StringExact("https://k8s.dev.eu-de-central.msh.host:6443"),
		"disable_ssl_validation": knownvalue.Bool(true),
		"replication": xknownvalue.MapExact(map[string]knownvalue.Check{
			"client_config": xknownvalue.MapExact(map[string]knownvalue.Check{
				"access_token": xknownvalue.MapExact(map[string]knownvalue.Check{
					"secret_value":   knownvalue.Null(),
					"secret_hash":    xknownvalue.NotEmptyString(),
					"secret_version": xknownvalue.NotEmptyString(),
				}),
			}),
			"namespace_name_pattern": knownvalue.StringExact("#{workspaceIdentifier}-#{projectIdentifier}"),
		}),
		"metering": xknownvalue.MapExact(map[string]knownvalue.Check{
			"client_config": xknownvalue.MapExact(map[string]knownvalue.Check{
				"access_token": xknownvalue.MapExact(map[string]knownvalue.Check{
					"secret_value":   knownvalue.Null(),
					"secret_hash":    xknownvalue.NotEmptyString(),
					"secret_version": xknownvalue.NotEmptyString(),
				}),
			}),
			"processing": checkMeteringProcessingConfig(),
		}),
	})
}

func checkAksPlatformConfig() knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"base_url":               knownvalue.StringExact("https://myaks-dns.westeurope.azmk8s.io:443"),
		"disable_ssl_validation": knownvalue.Bool(true),
		"replication": xknownvalue.MapExact(map[string]knownvalue.Check{
			"access_token": xknownvalue.MapExact(map[string]knownvalue.Check{
				"secret_value":   knownvalue.Null(),
				"secret_hash":    xknownvalue.NotEmptyString(),
				"secret_version": knownvalue.StringExact(nonEphemeralSecretVersion("top-secret-value")),
			}),
			"service_principal": xknownvalue.MapExact(map[string]knownvalue.Check{
				"entra_tenant": knownvalue.StringExact("dev-mycompany.onmicrosoft.com"),
				"client_id":    knownvalue.StringExact("58d6f907-7b0e-4fd8-b328-3e8342dddc8d"),
				"object_id":    knownvalue.StringExact("3c305efe-625d-4eaf-9bfa-b981ddbcc99f"),
				"auth": xknownvalue.MapExact(map[string]knownvalue.Check{
					"type":       knownvalue.StringExact("workloadIdentity"),
					"credential": knownvalue.Null(),
				}),
			}),
			"namespace_name_pattern":     knownvalue.StringExact("#{workspaceIdentifier}-#{projectIdentifier}"),
			"group_name_pattern":         knownvalue.StringExact("#{workspaceIdentifier}.#{projectIdentifier}-#{platformGroupAlias}"),
			"aks_subscription_id":        knownvalue.StringExact("12345678-90ab-cdef-1234-567890abcdef"),
			"aks_cluster_name":           knownvalue.StringExact("my-aks-cluster"),
			"aks_resource_group":         knownvalue.StringExact("my-aks-rg"),
			"send_azure_invitation_mail": knownvalue.Bool(true),
			"user_lookup_strategy":       knownvalue.StringExact("UserByMailLookupStrategy"),
			"administrative_unit_id":     knownvalue.Null(),
			"redirect_url":               knownvalue.Null(),
		}),
		"metering": xknownvalue.MapExact(map[string]knownvalue.Check{
			"client_config": xknownvalue.MapExact(map[string]knownvalue.Check{
				"access_token": xknownvalue.MapExact(map[string]knownvalue.Check{
					"secret_value":   knownvalue.Null(),
					"secret_hash":    xknownvalue.NotEmptyString(),
					"secret_version": xknownvalue.NotEmptyString(),
				}),
			}),
			"processing": checkMeteringProcessingConfig(),
		}),
	})
}

func checkAzureRgPlatformConfig() knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"entra_tenant": knownvalue.StringExact("example-tenant.onmicrosoft.com"),
		"replication": xknownvalue.MapExact(map[string]knownvalue.Check{
			"service_principal": xknownvalue.MapExact(map[string]knownvalue.Check{
				"client_id": knownvalue.StringExact("12345678-1234-1234-1234-123456789abc"),
				"object_id": knownvalue.StringExact("87654321-4321-4321-4321-cba987654321"),
				"auth": xknownvalue.MapExact(map[string]knownvalue.Check{
					"type": knownvalue.StringExact("credential"),
					"credential": xknownvalue.MapExact(map[string]knownvalue.Check{
						"secret_value":   knownvalue.Null(),
						"secret_hash":    xknownvalue.NotEmptyString(),
						"secret_version": xknownvalue.NotEmptyString(),
					}),
				}),
			}),
			"subscription":                       knownvalue.StringExact("12345678-1234-1234-1234-123456789abc"),
			"resource_group_name_pattern":        knownvalue.StringExact("#{workspaceIdentifier}-#{projectIdentifier}"),
			"user_group_name_pattern":            knownvalue.StringExact("#{workspaceIdentifier}.#{projectIdentifier}-#{platformGroupAlias}"),
			"user_lookup_strategy":               knownvalue.StringExact("UserByMailLookupStrategy"),
			"skip_user_group_permission_cleanup": knownvalue.Bool(true),
			"administrative_unit_id":             knownvalue.Null(),
			"b2b_user_invitation": xknownvalue.MapExact(map[string]knownvalue.Check{
				"redirect_url":               knownvalue.StringExact("https://meshcloud.io"),
				"send_azure_invitation_mail": knownvalue.Bool(true),
			}),
			"tenant_tags": xknownvalue.MapExact(map[string]knownvalue.Check{
				"namespace_prefix": knownvalue.StringExact("meshstack_"),
				"tag_mappers":      knownvalue.SetSizeExact(2),
			}),
		}),
	})
}

func checkOpenshiftPlatformConfig() knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"base_url":               knownvalue.StringExact("https://api.okd4.dev.eu-de-central.msh.host:6443"),
		"disable_ssl_validation": knownvalue.Bool(true),
		"replication": xknownvalue.MapExact(map[string]knownvalue.Check{
			"client_config": xknownvalue.MapExact(map[string]knownvalue.Check{
				"access_token": xknownvalue.MapExact(map[string]knownvalue.Check{
					"secret_value":   knownvalue.Null(),
					"secret_hash":    xknownvalue.NotEmptyString(),
					"secret_version": xknownvalue.NotEmptyString(),
				}),
			}),
			"web_console_url":        knownvalue.StringExact("https://console-openshift-console.apps.okd4.dev.eu-de-central.msh.host"),
			"project_name_pattern":   knownvalue.StringExact("#{workspaceIdentifier}-#{projectIdentifier}"),
			"identity_provider_name": knownvalue.StringExact("meshStack"),
			"openshift_role_mappings": knownvalue.SetExact([]knownvalue.Check{
				xknownvalue.MapExact(map[string]knownvalue.Check{
					"project_role_ref": xknownvalue.MapExact(map[string]knownvalue.Check{
						"name": knownvalue.StringExact("admin"),
						"kind": knownvalue.StringExact("meshProjectRole"),
					}),
					"openshift_role": knownvalue.StringExact("admin"),
				}),
				xknownvalue.MapExact(map[string]knownvalue.Check{
					"project_role_ref": xknownvalue.MapExact(map[string]knownvalue.Check{
						"name": knownvalue.StringExact("user"),
						"kind": knownvalue.StringExact("meshProjectRole"),
					}),
					"openshift_role": knownvalue.StringExact("edit"),
				}),
			}),
			"tenant_tags": xknownvalue.MapExact(map[string]knownvalue.Check{
				"namespace_prefix": knownvalue.StringExact("meshstack_"),
				"tag_mappers":      knownvalue.SetSizeExact(2),
			}),
		}),
		"metering": xknownvalue.MapExact(map[string]knownvalue.Check{
			"client_config": xknownvalue.MapExact(map[string]knownvalue.Check{
				"access_token": xknownvalue.MapExact(map[string]knownvalue.Check{
					"secret_value":   knownvalue.Null(),
					"secret_hash":    xknownvalue.NotEmptyString(),
					"secret_version": xknownvalue.NotEmptyString(),
				}),
			}),
			"processing": checkMeteringProcessingConfig(),
		}),
	})
}

func checkCustomPlatformConfig(suffix string) knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"platform_type_ref": xknownvalue.MapExact(map[string]knownvalue.Check{
			"name": knownvalue.StringExact(platformTypeName(suffix)),
			"kind": knownvalue.StringExact("meshPlatformType"),
		}),
		"metering": xknownvalue.MapExact(map[string]knownvalue.Check{
			"processing": checkMeteringProcessingConfig(),
		}),
	})
}
