package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccPlatformResource(t *testing.T) {
	t.Parallel()

	t.Run("01_azure", func(t *testing.T) {
		config, resourceAddress := testconfig.BuildPlatformConfig(t, "_01_azure")
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
					ConfigStateChecks: append(
						[]statecheck.StateCheck{
							statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("metadata"), checkPlatformMetadata(&resourceUuid)),
							statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("Example Platform")),
							statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("identifier"), knownvalue.StringFunc(func(value string) error {
								parts := strings.SplitN(value, ".", 2)
								if len(parts) != 2 || !strings.HasPrefix(parts[0], "my-platform-") || parts[1] == "" {
									return fmt.Errorf("expected identifier format <platform>.<location>, got %q", value)
								}
								return nil
							})),
						},
						checkPlatformConfigState(resourceAddress.String(), "01_azure")...,
					),
				},
				{
					Config: func() string {
						u := config.WithFirstBlock(t,
							testconfig.Traverse(t, "spec", "display_name")(testconfig.SetString("Example Platform Updated")))
						return u.String()
					}(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("Example Platform Updated")),
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

	t.Run("02_aws", func(t *testing.T) {
		config, resourceAddress := testconfig.BuildPlatformConfig(t, "_02_aws")
		var resourceUuid string
		ApplyAndTest(t, resource.TestCase{
			Steps: platformCreateImportSteps(config, resourceAddress, &resourceUuid, "02_aws"),
		})
	})

	t.Run("03_gcp", func(t *testing.T) {
		config, resourceAddress := testconfig.BuildPlatformConfig(t, "_03_gcp")
		var resourceUuid string
		ApplyAndTest(t, resource.TestCase{
			Steps: platformCreateImportSteps(config, resourceAddress, &resourceUuid, "03_gcp"),
		})
	})

	t.Run("04_kubernetes", func(t *testing.T) {
		config, resourceAddress := testconfig.BuildPlatformConfig(t, "_04_kubernetes")
		var resourceUuid string
		ApplyAndTest(t, resource.TestCase{
			Steps: platformCreateImportSteps(config, resourceAddress, &resourceUuid, "04_kubernetes"),
		})
	})

	t.Run("05_aks", func(t *testing.T) {
		config, resourceAddress := testconfig.BuildPlatformConfig(t, "_05_aks")
		var resourceUuid string
		importStep := resource.TestStep{
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
					plancheck.ExpectUnknownValue(resourceAddress.String(), aksSecretPath().AtMapKey("secret_hash")),
					plancheck.ExpectKnownValue(resourceAddress.String(), aksSecretPath().AtMapKey("secret_version"), knownvalue.StringExact("4823648dbe986627638418ba4469261474bd52043ffef910a5b2d62c92df86bc")),
				},
			},
		}
		ApplyAndTest(t, resource.TestCase{
			Steps: append(
				platformCreateSteps(config, resourceAddress, &resourceUuid, "05_aks"),
				importStep,
			),
		})
	})

	t.Run("06_azurerg", func(t *testing.T) {
		config, resourceAddress := testconfig.BuildPlatformConfig(t, "_06_azurerg")
		var resourceUuid string
		ApplyAndTest(t, resource.TestCase{
			Steps: platformCreateImportSteps(config, resourceAddress, &resourceUuid, "06_azurerg"),
		})
	})

	t.Run("07_openshift", func(t *testing.T) {
		config, resourceAddress := testconfig.BuildPlatformConfig(t, "_07_openshift")
		var resourceUuid string
		ApplyAndTest(t, resource.TestCase{
			Steps: platformCreateImportSteps(config, resourceAddress, &resourceUuid, "07_openshift"),
		})
	})

	t.Run("08_custom", func(t *testing.T) {
		config, resourceAddress, _ := testconfig.BuildCustomPlatformAndWorkspaceConfig(t)
		var resourceUuid string
		ApplyAndTest(t, resource.TestCase{
			Steps: platformCreateImportSteps(config, resourceAddress, &resourceUuid, "08_custom"),
		})
	})
}

// aksSecretPath is a factory to work around slice copy/clone bug in tfjsonpath.Path.AtMapKey.
func aksSecretPath() tfjsonpath.Path {
	return tfjsonpath.New("spec").AtMapKey("config").AtMapKey("aks").AtMapKey("replication").AtMapKey("access_token")
}

// platformCreateSteps returns create+state-check steps for a platform test.
func platformCreateSteps(config testconfig.Config, resourceAddress testconfig.Traversal, resourceUuidOut *string, exampleSuffix string) []resource.TestStep {
	return []resource.TestStep{
		{
			Config: config.String(),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(resourceAddress.String(), plancheck.ResourceActionCreate),
				},
			},
			ConfigStateChecks: append(
				[]statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("metadata"), checkPlatformMetadata(resourceUuidOut)),
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("Example Platform")),
				},
				checkPlatformConfigState(resourceAddress.String(), exampleSuffix)...,
			),
		},
	}
}

// platformCreateImportSteps returns create + import steps for a platform test.
func platformCreateImportSteps(config testconfig.Config, resourceAddress testconfig.Traversal, resourceUuidOut *string, exampleSuffix string) []resource.TestStep {
	return append(
		platformCreateSteps(config, resourceAddress, resourceUuidOut, exampleSuffix),
		resource.TestStep{
			ImportState:     true,
			ImportStateKind: resource.ImportBlockWithID,
			ImportStateIdFunc: func(state *terraform.State) (string, error) {
				return *resourceUuidOut, nil
			},
			ResourceName: resourceAddress.String(),
		},
	)
}

// PlatformResourceConfigForTest creates a workspace + Azure platform config for data source testing.
// If resourceAddress is non-nil, it is populated with the platform resource address.
// If platformUuidOut is non-nil, state capture is set up via checkPlatformMetadata.
func PlatformResourceConfigForTest(t *testing.T, resourceAddress *testconfig.Traversal, platformUuidOut *string) testconfig.Config {
	t.Helper()
	workspaceConfig, workspaceAddr := testconfig.BuildWorkspaceConfig(t)

	platformSuffix := acctest.RandString(8)
	platformConfig := (testconfig.Resource{Name: "platform", Suffix: "_01_azure"}).Config(t)
	var platformAddr testconfig.Traversal
	platformConfig = platformConfig.WithFirstBlock(t,
		testconfig.ExtractIdentifier(&platformAddr),
		testconfig.Traverse(t, "metadata", "name")(testconfig.SetString(fmt.Sprintf("my-platform-%s", platformSuffix))))
	platformConfig = platformConfig.WithFirstBlock(t, testconfig.OwnedByWorkspace(t, workspaceAddr))

	if resourceAddress != nil {
		*resourceAddress = platformAddr
	}

	return platformConfig.Join(workspaceConfig)
}

func checkPlatformMetadata(resourceUuidOut *string) knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"name":               xknownvalue.NotEmptyString(),
		"owned_by_workspace": xknownvalue.NotEmptyString(),
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

func checkPlatformConfigState(resourceAddress, exampleSuffix string) []statecheck.StateCheck {
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
		configCheck = checkCustomPlatformConfig()
	default:
		platformType = "azure"
		configCheck = knownvalue.NotNull()
	}

	return []statecheck.StateCheck{
		statecheck.ExpectKnownValue(
			resourceAddress,
			tfjsonpath.New("spec").AtMapKey("config").AtMapKey(platformType),
			configCheck,
		),
		checkPlatformQuotas(resourceAddress, exampleSuffix),
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
			"web_console_url":               knownvalue.StringExact("https://console-openshift-console.apps.okd4.dev.eu-de-central.msh.host"),
			"project_name_pattern":          knownvalue.StringExact("#{workspaceIdentifier}-#{projectIdentifier}"),
			"enable_template_instantiation": knownvalue.Bool(true),
			"identity_provider_name":        knownvalue.StringExact("meshStack"),
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

func checkCustomPlatformConfig() knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"platform_type_ref": xknownvalue.MapExact(map[string]knownvalue.Check{
			"name": xknownvalue.NotEmptyString(),
			"kind": knownvalue.StringExact("meshPlatformType"),
		}),
		"metering": xknownvalue.MapExact(map[string]knownvalue.Check{
			"processing": checkMeteringProcessingConfig(),
		}),
	})
}
