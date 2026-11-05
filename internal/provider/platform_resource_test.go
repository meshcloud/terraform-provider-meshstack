package provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccPlatformResource(t *testing.T) {
	t.Parallel()

	t.Run("01_azure", func(t *testing.T) {
		suffix := acctest.RandString(8)
		vars := hclStepVars{"suffix": suffix}
		resourceAddress := platformAddress("01_azure")
		var resourceUuid string
		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/platform/01_azure", 1, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: append(
						[]statecheck.StateCheck{
							statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata"), checkPlatformMetadata("test-platform-"+suffix, "test-ws-"+suffix, &resourceUuid)),
							statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("Example Platform")),
							statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("identifier"), knownvalue.StringExact("test-platform-"+suffix+".meshcloud-azure-dev")),
							statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("access_information"), knownvalue.StringExact("Login via [Azure Portal](https://portal.azure.com) using your corporate credentials.")),
						},
						checkPlatformConfigState(resourceAddress, "01_azure", suffix)...,
					),
				},
				{
					Config: hclStep(t, "resources/platform/01_azure", 2, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("Example Platform Updated")),
					},
				},
				platformExampleStep(t, "01_azure", 3, suffix),
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName: resourceAddress,
				},
			},
		})
	})

	t.Run("02_aws", func(t *testing.T) {
		var resourceUuid string
		ApplyAndTest(t, resource.TestCase{
			Steps: platformCreateImportSteps(t, "02_aws", acctest.RandString(8), &resourceUuid),
		})
	})

	t.Run("03_gcp", func(t *testing.T) {
		var resourceUuid string
		ApplyAndTest(t, resource.TestCase{
			Steps: platformCreateImportSteps(t, "03_gcp", acctest.RandString(8), &resourceUuid),
		})
	})

	t.Run("04_kubernetes", func(t *testing.T) {
		var resourceUuid string
		ApplyAndTest(t, resource.TestCase{
			Steps: platformCreateImportSteps(t, "04_kubernetes", acctest.RandString(8), &resourceUuid),
		})
	})

	t.Run("05_aks", func(t *testing.T) {
		suffix := acctest.RandString(8)
		resourceAddress := platformAddress("05_aks")
		var resourceUuid string
		importStep := resource.TestStep{
			ImportState:     true,
			ImportStateKind: resource.ImportBlockWithID,
			ImportStateIdFunc: func(state *terraform.State) (string, error) {
				return resourceUuid, nil
			},
			ResourceName:       resourceAddress,
			ExpectNonEmptyPlan: true,
			ImportPlanChecks: resource.ImportPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionUpdate),
					plancheck.ExpectUnknownValue(resourceAddress, aksSecretPath().AtMapKey("secret_hash")),
					plancheck.ExpectKnownValue(resourceAddress, aksSecretPath().AtMapKey("secret_version"), knownvalue.StringExact("4823648dbe986627638418ba4469261474bd52043ffef910a5b2d62c92df86bc")),
				},
			},
		}
		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				platformCreateStep(t, "05_aks", suffix, &resourceUuid),
				platformExampleStep(t, "05_aks", 2, suffix),
				importStep,
			},
		})
	})

	t.Run("06_azurerg", func(t *testing.T) {
		var resourceUuid string
		ApplyAndTest(t, resource.TestCase{
			Steps: platformCreateImportSteps(t, "06_azurerg", acctest.RandString(8), &resourceUuid),
		})
	})

	t.Run("07_openshift", func(t *testing.T) {
		var resourceUuid string
		ApplyAndTest(t, resource.TestCase{
			Steps: platformCreateImportSteps(t, "07_openshift", acctest.RandString(8), &resourceUuid),
		})
	})

	t.Run("08_custom", func(t *testing.T) {
		var resourceUuid string
		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				platformCreateStep(t, "08_custom", acctest.RandString(8), &resourceUuid),
				platformImportStep("08_custom", &resourceUuid),
			},
		})
	})
}

// aksSecretPath is a factory to work around slice copy/clone bug in tfjsonpath.Path.AtMapKey.
func aksSecretPath() tfjsonpath.Path {
	return tfjsonpath.New("spec").AtMapKey("config").AtMapKey("aks").AtMapKey("replication").AtMapKey("access_token")
}

func platformKind(exampleSuffix string) string {
	_, kind, _ := strings.Cut(exampleSuffix, "_")
	return kind
}

func platformAddress(exampleSuffix string) string {
	return "meshstack_platform.example_" + platformKind(exampleSuffix)
}

func platformCreateStep(t *testing.T, exampleSuffix, suffix string, resourceUuidOut *string) resource.TestStep {
	t.Helper()
	resourceAddress := platformAddress(exampleSuffix)
	return resource.TestStep{
		Config: hclStep(t, "resources/platform/"+exampleSuffix, 1, platformTypeStepVars(suffix)),
		ConfigPlanChecks: resource.ConfigPlanChecks{
			PreApply: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionCreate),
			},
		},
		ConfigStateChecks: append(
			[]statecheck.StateCheck{
				statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata"), checkPlatformMetadata("test-platform-"+suffix, "test-ws-"+suffix, resourceUuidOut)),
				statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("Example Platform")),
			},
			checkPlatformConfigState(resourceAddress, exampleSuffix, suffix)...,
		),
	}
}

func platformExampleStep(t *testing.T, exampleSuffix string, step int, suffix string) resource.TestStep {
	t.Helper()
	resourceAddress := platformAddress(exampleSuffix)
	return resource.TestStep{
		Config: hclStep(t, "resources/platform/"+exampleSuffix, step, platformTypeStepVars(suffix)),
		ConfigPlanChecks: resource.ConfigPlanChecks{
			PreApply: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionUpdate),
			},
		},
	}
}

func platformImportStep(exampleSuffix string, resourceUuidOut *string) resource.TestStep {
	resourceAddress := platformAddress(exampleSuffix)
	return resource.TestStep{
		ImportState:     true,
		ImportStateKind: resource.ImportBlockWithID,
		ImportStateIdFunc: func(state *terraform.State) (string, error) {
			return *resourceUuidOut, nil
		},
		ResourceName: resourceAddress,
	}
}

func platformCreateImportSteps(t *testing.T, exampleSuffix, suffix string, resourceUuidOut *string) []resource.TestStep {
	t.Helper()
	return []resource.TestStep{
		platformCreateStep(t, exampleSuffix, suffix, resourceUuidOut),
		platformExampleStep(t, exampleSuffix, 2, suffix),
		platformImportStep(exampleSuffix, resourceUuidOut),
	}
}

func checkPlatformMetadata(name, workspace string, resourceUuidOut *string) knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"name":               knownvalue.StringExact(name),
		"owned_by_workspace": knownvalue.StringExact(workspace),
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
		configCheck = checkCustomPlatformConfig("TEST-PT-" + strings.ToUpper(suffix))
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
		checkPlatformQuotas(resourceAddress),
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

func checkPlatformQuotas(resourceAddress string) statecheck.StateCheck {
	return statecheck.ExpectKnownValue(
		resourceAddress,
		tfjsonpath.New("spec").AtMapKey("quota_definitions"),
		knownvalue.SetSizeExact(0),
	)
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
				"secret_version": xknownvalue.NotEmptyString(),
			}),
			"service_principal": xknownvalue.MapExact(map[string]knownvalue.Check{
				"entra_tenant": knownvalue.StringExact("dev-mycompany.onmicrosoft.com"),
				"client_id":    xknownvalue.NotEmptyString(),
				"object_id":    xknownvalue.NotEmptyString(),
				"auth": xknownvalue.MapExact(map[string]knownvalue.Check{
					"type":       knownvalue.StringExact("workloadIdentity"),
					"credential": knownvalue.Null(),
				}),
			}),
			"namespace_name_pattern":     knownvalue.StringExact("#{workspaceIdentifier}-#{projectIdentifier}"),
			"group_name_pattern":         knownvalue.StringExact("#{workspaceIdentifier}.#{projectIdentifier}-#{platformGroupAlias}"),
			"aks_subscription_id":        xknownvalue.NotEmptyString(),
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
				"client_id": xknownvalue.NotEmptyString(),
				"object_id": xknownvalue.NotEmptyString(),
				"auth": xknownvalue.MapExact(map[string]knownvalue.Check{
					"type": knownvalue.StringExact("credential"),
					"credential": xknownvalue.MapExact(map[string]knownvalue.Check{
						"secret_value":   knownvalue.Null(),
						"secret_hash":    xknownvalue.NotEmptyString(),
						"secret_version": xknownvalue.NotEmptyString(),
					}),
				}),
			}),
			"subscription":                       xknownvalue.NotEmptyString(),
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

func checkCustomPlatformConfig(platformTypeName string) knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"platform_type_ref": xknownvalue.MapExact(map[string]knownvalue.Check{
			"name": knownvalue.StringExact(platformTypeName),
			"kind": knownvalue.StringExact("meshPlatformType"),
		}),
		"metering": xknownvalue.MapExact(map[string]knownvalue.Check{
			"processing": checkMeteringProcessingConfig(),
		}),
	})
}
