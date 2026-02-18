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
	"github.com/stretchr/testify/assert"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/clientmock"
	"github.com/meshcloud/terraform-provider-meshstack/internal/util/xknownvalue"
)

func TestAccPlatformResource(t *testing.T) {
	runPlatformTestCases(t)
}

func TestPlatformResource(t *testing.T) {
	runPlatformTestCases(t, SetupMockClient(func(t *testing.T, testCase *resource.TestCase, mockClient clientmock.Client) {
		t.Helper()
		testCase.Steps[0].PostApplyFunc = func() {
			assert.Len(t, mockClient.Platform.Store, 1)
		}
	}))
}

func runPlatformTestCases(t *testing.T, modifiers ...ResourceTestCaseModifier) {
	t.Helper()
	platformExamples := examples.Resource{Name: "platform"}
	for exampleResource := range platformExamples.All() {
		exampleSuffix := strings.TrimPrefix(exampleResource.Suffix, "_")
		t.Run(exampleSuffix, func(t *testing.T) {
			t.Parallel()
			var resourceAddress, nameSuffixAddress examples.Identifier
			config := exampleResource.Config().
				SingleResourceAddress(&resourceAddress).
				OwnedByAdminWorkspace().
				Join(
					platformExamples.TestSupportConfig("_random").SingleResourceAddress(&nameSuffixAddress),
				).ReplaceAll(`name               = "my-platform"`, nameSuffixAddress.Format(`name = "my-platform-${%s.result}"`))

			if exampleSuffix == "08_custom" {
				var platformTypeAddress examples.Identifier
				config = config.Join(
					exampleResource.TestSupportConfig("_platform_type").
						OwnedByAdminWorkspace().
						SingleResourceAddressWithType("meshstack_platform_type", &platformTypeAddress),
				).ReplaceAll(`platform_type_ref = { name = "my-custom-platform-type" }`, platformTypeAddress.Format(`platform_type_ref = %s.ref`))
			}

			var resourceUuid string
			testSteps := []resource.TestStep{
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
						},
						checkPlatformConfigState(resourceAddress.String(), exampleSuffix)...,
					),
				},
			}
			// Only test update with Azure example to keep tests fast
			if exampleSuffix == "01_azure" {
				testSteps = append(testSteps, resource.TestStep{
					Config: config.ReplaceAll(`display_name      = "Example Platform"`, `display_name      = "Example Platform Updated"`).String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(resourceAddress.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("Example Platform Updated")),
					},
				})
			}

			testSteps = append(testSteps,
				resource.TestStep{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName: resourceAddress.String(),
				},
			)

			ResourceTestCaseModifiers(modifiers).ApplyAndTest(t, resource.TestCase{Steps: testSteps})
		})
	}
}

func PlatformResourceConfigForTest(resourceAddress, platformName *examples.Identifier) examples.Config {
	// Use the first example (Azure) as a representative example for data source testing
	platformExamples := examples.Resource{Name: "platform"}
	config := (examples.Resource{Name: "platform", Suffix: "_01_azure"}).Config().
		OwnedByAdminWorkspace()

	if resourceAddress != nil {
		config = config.SingleResourceAddress(resourceAddress)
	}

	if platformName != nil {
		var nameSuffixAddress examples.Identifier
		config = config.Join(
			platformExamples.TestSupportConfig("_random").SingleResourceAddress(&nameSuffixAddress),
		).ReplaceAll(`name               = "my-platform"`, nameSuffixAddress.Format(`name = "my-platform-${%s.result}"`))
		*platformName = nameSuffixAddress
	}

	return config
}

func checkPlatformMetadata(resourceUuidOut *string) knownvalue.Check {
	return knownvalue.MapExact(map[string]knownvalue.Check{
		"name":               KnownValueNotEmptyString(),
		"owned_by_workspace": knownvalue.StringExact("managed-customer"),
		"uuid": KnownValueNotEmptyString(func(actualValue string) error {
			*resourceUuidOut = actualValue
			return nil
		}),
	})
}

func checkMeteringProcessingConfig() knownvalue.Check {
	return knownvalue.MapExact(map[string]knownvalue.Check{
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
		return statecheck.ExpectKnownValue(
			resourceAddress,
			tfjsonpath.New("spec").AtMapKey("quota_definitions"),
			knownvalue.SetExact([]knownvalue.Check{
				knownvalue.MapExact(map[string]knownvalue.Check{
					"quota_key":               knownvalue.StringExact("vcpu"),
					"label":                   knownvalue.StringExact("Virtual CPUs"),
					"description":             knownvalue.StringExact("Number of virtual CPUs available"),
					"unit":                    knownvalue.StringExact("cores"),
					"min_value":               knownvalue.Int64Exact(0),
					"max_value":               knownvalue.Int64Exact(100),
					"auto_approval_threshold": knownvalue.Int64Exact(50),
				}),
				knownvalue.MapExact(map[string]knownvalue.Check{
					"quota_key":               knownvalue.StringExact("storage"),
					"label":                   knownvalue.StringExact("Storage"),
					"description":             knownvalue.StringExact("Storage capacity in GB"),
					"unit":                    knownvalue.StringExact("GB"),
					"min_value":               knownvalue.Int64Exact(0),
					"max_value":               knownvalue.Int64Exact(1000),
					"auto_approval_threshold": knownvalue.Int64Exact(500),
				}),
			}),
		)
	default:
		// Other examples don't have quotas, just ensure the field exists
		return statecheck.ExpectKnownValue(
			resourceAddress,
			tfjsonpath.New("spec").AtMapKey("quota_definitions"),
			knownvalue.SetSizeExact(0),
		)
	}
}

func checkAzurePlatformConfig() knownvalue.Check {
	return knownvalue.MapExact(map[string]knownvalue.Check{
		"entra_tenant": KnownValueNotEmptyString(),
		"replication": xknownvalue.MapExact(map[string]knownvalue.Check{
			"service_principal": knownvalue.MapExact(map[string]knownvalue.Check{
				"client_id": KnownValueNotEmptyString(),
				"object_id": KnownValueNotEmptyString(),
				"auth": knownvalue.MapExact(map[string]knownvalue.Check{
					"type":       knownvalue.StringExact("workloadIdentity"),
					"credential": knownvalue.Null(),
				}),
			}),
			"update_subscription_name": knownvalue.Bool(false),
			"provisioning": knownvalue.MapExact(map[string]knownvalue.Check{
				"subscription_owner_object_ids": knownvalue.SetExact([]knownvalue.Check{
					KnownValueNotEmptyString(),
				}),
				"enterprise_enrollment": knownvalue.Null(),
				"customer_agreement":    knownvalue.Null(),
				"pre_provisioned": knownvalue.MapExact(map[string]knownvalue.Check{
					"unused_subscription_name_prefix": knownvalue.StringExact("unused-"),
				}),
			}),
			"b2b_user_invitation": knownvalue.MapExact(map[string]knownvalue.Check{
				"redirect_url":               knownvalue.StringExact("https://portal.azure.com/#home"),
				"send_azure_invitation_mail": knownvalue.Bool(false),
			}),
			"subscription_name_pattern":   knownvalue.StringExact("#{workspaceIdentifier}.#{projectIdentifier}"),
			"group_name_pattern":          knownvalue.StringExact("#{workspaceIdentifier}.#{projectIdentifier}-#{platformGroupAlias}"),
			"blueprint_service_principal": KnownValueNotEmptyString(),
			"blueprint_location":          knownvalue.StringExact("westeurope"),
			"azure_role_mappings": knownvalue.SetExact([]knownvalue.Check{
				knownvalue.MapExact(map[string]knownvalue.Check{
					"project_role_ref": knownvalue.MapExact(map[string]knownvalue.Check{
						"name": knownvalue.StringExact("admin"),
						"kind": knownvalue.StringExact("meshProjectRole"),
					}),
					"azure_role": knownvalue.MapExact(map[string]knownvalue.Check{
						"alias": knownvalue.StringExact("admin"),
						"id":    KnownValueNotEmptyString(),
					}),
				}),
				knownvalue.MapExact(map[string]knownvalue.Check{
					"project_role_ref": knownvalue.MapExact(map[string]knownvalue.Check{
						"name": knownvalue.StringExact("reader"),
						"kind": knownvalue.StringExact("meshProjectRole"),
					}),
					"azure_role": knownvalue.MapExact(map[string]knownvalue.Check{
						"alias": knownvalue.StringExact("reader"),
						"id":    KnownValueNotEmptyString(),
					}),
				}),
				knownvalue.MapExact(map[string]knownvalue.Check{
					"project_role_ref": knownvalue.MapExact(map[string]knownvalue.Check{
						"name": knownvalue.StringExact("user"),
						"kind": knownvalue.StringExact("meshProjectRole"),
					}),
					"azure_role": knownvalue.MapExact(map[string]knownvalue.Check{
						"alias": knownvalue.StringExact("user"),
						"id":    KnownValueNotEmptyString(),
					}),
				}),
			}),
			"tenant_tags": knownvalue.MapExact(map[string]knownvalue.Check{
				"namespace_prefix": knownvalue.StringExact("meshstack_"),
				"tag_mappers":      knownvalue.SetSizeExact(7),
			}),
			"user_lookup_strategy":                           knownvalue.StringExact("UserByMailLookupStrategy"),
			"skip_user_group_permission_cleanup":             knownvalue.Bool(false),
			"allow_hierarchical_management_group_assignment": knownvalue.Bool(false),
			"administrative_unit_id":                         knownvalue.Null(),
		}),
		"metering": knownvalue.Null(),
	})
}

func checkAwsPlatformConfig() knownvalue.Check {
	return knownvalue.MapExact(map[string]knownvalue.Check{
		"region": knownvalue.StringExact("us-east-1"),
		"replication": xknownvalue.MapExact(map[string]knownvalue.Check{
			"access_config": knownvalue.MapExact(map[string]knownvalue.Check{
				"organization_root_account_role":        knownvalue.StringExact("OrganizationAccountAccessRole"),
				"organization_root_account_external_id": knownvalue.Null(),
				"auth": knownvalue.MapExact(map[string]knownvalue.Check{
					"type": knownvalue.StringExact("credential"),
					"credential": knownvalue.MapExact(map[string]knownvalue.Check{
						"access_key": knownvalue.StringExact("AKIAIOSFODNN7EXAMPLE"),
						"secret_key": knownvalue.MapExact(map[string]knownvalue.Check{
							"secret_value":   knownvalue.Null(),
							"secret_hash":    KnownValueNotEmptyString(),
							"secret_version": KnownValueNotEmptyString(),
						}),
					}),
					"workload_identity": knownvalue.Null(),
				}),
			}),
			"account_alias_pattern":                             knownvalue.StringExact("#{workspaceIdentifier}-#{projectIdentifier}"),
			"account_email_pattern":                             knownvalue.StringExact("aws+#{workspaceIdentifier}.#{projectIdentifier}@example.com"),
			"automation_account_role":                           knownvalue.StringExact("OrganizationAccountAccessRole"),
			"automation_account_external_id":                    knownvalue.Null(),
			"account_access_role":                               knownvalue.StringExact("OrganizationAccountAccessRole"),
			"self_downgrade_access_role":                        knownvalue.Bool(false),
			"enforce_account_alias":                             knownvalue.Bool(false),
			"wait_for_external_avm":                             knownvalue.Bool(false),
			"skip_user_group_permission_cleanup":                knownvalue.Bool(false),
			"allow_hierarchical_organizational_unit_assignment": knownvalue.Bool(false),
			"enrollment_configuration":                          knownvalue.Null(),
			"aws_sso": xknownvalue.MapExact(map[string]knownvalue.Check{
				"arn":                knownvalue.StringExact("arn:aws:sso:::instance/ssoins-1234567890abcdef"),
				"scim_endpoint":      knownvalue.StringExact("https://scim.us-east-1.amazonaws.com/abcd1234-5678-90ab-cdef-example12345/scim/v2/"),
				"group_name_pattern": knownvalue.StringExact("#{workspaceIdentifier}.#{projectIdentifier}-#{platformGroupAlias}"),
				"sso_access_token": knownvalue.MapExact(map[string]knownvalue.Check{
					"secret_value":   knownvalue.Null(),
					"secret_hash":    KnownValueNotEmptyString(),
					"secret_version": KnownValueNotEmptyString(),
				}),
				"sign_in_url": knownvalue.StringExact("https://my-sso-portal.awsapps.com/start"),
				"aws_role_mappings": knownvalue.ListExact([]knownvalue.Check{
					knownvalue.MapExact(map[string]knownvalue.Check{
						"project_role_ref": knownvalue.MapExact(map[string]knownvalue.Check{
							"name": knownvalue.StringExact("admin"),
							"kind": knownvalue.StringExact("meshProjectRole"),
						}),
						"aws_role":            knownvalue.StringExact("admin"),
						"permission_set_arns": knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("arn:aws:sso:::permissionSet/ssoins-1234567890abcdef/ps-1234567890abcdef")}),
					}),
				}),
			}),
			"tenant_tags": knownvalue.MapExact(map[string]knownvalue.Check{
				"namespace_prefix": knownvalue.StringExact("meshstack_"),
				"tag_mappers":      knownvalue.SetSizeExact(2),
			}),
		}),
		"metering": knownvalue.MapExact(map[string]knownvalue.Check{
			"access_config": knownvalue.MapExact(map[string]knownvalue.Check{
				"organization_root_account_role":        knownvalue.StringExact("OrganizationAccountAccessRole"),
				"organization_root_account_external_id": knownvalue.Null(),
				"auth": knownvalue.MapExact(map[string]knownvalue.Check{
					"type": knownvalue.StringExact("credential"),
					"credential": knownvalue.MapExact(map[string]knownvalue.Check{
						"access_key": knownvalue.StringExact("AKIAIOSFODNN7EXAMPLE"),
						"secret_key": knownvalue.MapExact(map[string]knownvalue.Check{
							"secret_value":   knownvalue.Null(),
							"secret_hash":    KnownValueNotEmptyString(),
							"secret_version": KnownValueNotEmptyString(),
						}),
					}),
					"workload_identity": knownvalue.Null(),
				}),
			}),
			"filter":                            knownvalue.StringExact("NONE"),
			"reserved_instance_fair_chargeback": knownvalue.Bool(false),
			"savings_plan_fair_chargeback":      knownvalue.Bool(false),
			"processing":                        checkMeteringProcessingConfig(),
		}),
	})
}

func checkGcpPlatformConfig() knownvalue.Check {
	return knownvalue.MapExact(map[string]knownvalue.Check{
		"replication": knownvalue.MapExact(map[string]knownvalue.Check{
			"service_account": knownvalue.MapExact(map[string]knownvalue.Check{
				"type": knownvalue.StringExact("credential"),
				"credential": knownvalue.MapExact(map[string]knownvalue.Check{
					"secret_value":   knownvalue.Null(),
					"secret_hash":    KnownValueNotEmptyString(),
					"secret_version": KnownValueNotEmptyString(),
				}),
				"workload_identity": knownvalue.Null(),
			}),
			"project_id_pattern":                   knownvalue.StringExact("#{workspaceIdentifier}-#{projectIdentifier}"),
			"project_name_pattern":                 knownvalue.StringExact("#{workspaceIdentifier}.#{projectIdentifier}"),
			"group_name_pattern":                   knownvalue.StringExact("#{workspaceIdentifier}.#{projectIdentifier}-#{platformGroupAlias}"),
			"billing_account_id":                   knownvalue.StringExact("012345-6789AB-CDEF01"),
			"domain":                               knownvalue.StringExact("example.com"),
			"customer_id":                          knownvalue.StringExact("C01234567"),
			"user_lookup_strategy":                 knownvalue.StringExact("email"),
			"used_external_id_type":                knownvalue.Null(),
			"allow_hierarchical_folder_assignment": knownvalue.Bool(false),
			"skip_user_group_permission_cleanup":   knownvalue.Bool(false),
			"gcp_role_mappings": knownvalue.ListExact([]knownvalue.Check{
				knownvalue.MapExact(map[string]knownvalue.Check{
					"gcp_role": knownvalue.StringExact("roles/editor"),
					"project_role_ref": knownvalue.MapExact(map[string]knownvalue.Check{
						"kind": knownvalue.StringExact("meshProjectRole"),
						"name": knownvalue.StringExact("admin"),
					}),
				}),
				knownvalue.MapExact(map[string]knownvalue.Check{
					"gcp_role": knownvalue.StringExact("roles/viewer"),
					"project_role_ref": knownvalue.MapExact(map[string]knownvalue.Check{
						"kind": knownvalue.StringExact("meshProjectRole"),
						"name": knownvalue.StringExact("reader"),
					}),
				}),
			}),
			"tenant_tags": knownvalue.MapExact(map[string]knownvalue.Check{
				"namespace_prefix": knownvalue.StringExact("meshstack_"),
				"tag_mappers":      knownvalue.SetSizeExact(2),
			}),
		}),
		"metering": knownvalue.MapExact(map[string]knownvalue.Check{
			"service_account": knownvalue.MapExact(map[string]knownvalue.Check{
				"type": knownvalue.StringExact("credential"),
				"credential": knownvalue.MapExact(map[string]knownvalue.Check{
					"secret_value":   knownvalue.Null(),
					"secret_hash":    KnownValueNotEmptyString(),
					"secret_version": KnownValueNotEmptyString(),
				}),
				"workload_identity": knownvalue.Null(),
			}),
			"bigquery_table":                               knownvalue.StringExact("gcp_billing_export_v1"),
			"bigquery_table_for_carbon_footprint":          knownvalue.Null(),
			"carbon_footprint_data_collection_start_month": knownvalue.Null(),
			"partition_time_column":                        knownvalue.StringExact("usage_start_time"),
			"additional_filter":                            knownvalue.Null(),
			"processing":                                   checkMeteringProcessingConfig(),
		}),
	})
}

func checkKubernetesPlatformConfig() knownvalue.Check {
	return knownvalue.MapExact(map[string]knownvalue.Check{
		"base_url":               knownvalue.StringExact("https://k8s.dev.eu-de-central.msh.host:6443"),
		"disable_ssl_validation": knownvalue.Bool(false),
		"replication": knownvalue.MapExact(map[string]knownvalue.Check{
			"client_config": knownvalue.MapExact(map[string]knownvalue.Check{
				"access_token": knownvalue.MapExact(map[string]knownvalue.Check{
					"secret_value":   knownvalue.Null(),
					"secret_hash":    KnownValueNotEmptyString(),
					"secret_version": KnownValueNotEmptyString(),
				}),
			}),
			"namespace_name_pattern": knownvalue.StringExact("#{workspaceIdentifier}-#{projectIdentifier}"),
		}),
		"metering": knownvalue.MapExact(map[string]knownvalue.Check{
			"client_config": knownvalue.MapExact(map[string]knownvalue.Check{
				"access_token": knownvalue.MapExact(map[string]knownvalue.Check{
					"secret_value":   knownvalue.Null(),
					"secret_hash":    KnownValueNotEmptyString(),
					"secret_version": KnownValueNotEmptyString(),
				}),
			}),
			"processing": checkMeteringProcessingConfig(),
		}),
	})
}

func checkAksPlatformConfig() knownvalue.Check {
	return knownvalue.MapExact(map[string]knownvalue.Check{
		"base_url":               knownvalue.StringExact("https://myaks-dns.westeurope.azmk8s.io:443"),
		"disable_ssl_validation": knownvalue.Bool(false),
		"replication": xknownvalue.MapExact(map[string]knownvalue.Check{
			"access_token": knownvalue.MapExact(map[string]knownvalue.Check{
				"secret_value":   knownvalue.Null(),
				"secret_hash":    KnownValueNotEmptyString(),
				"secret_version": KnownValueNotEmptyString(),
			}),
			"service_principal": knownvalue.MapExact(map[string]knownvalue.Check{
				"entra_tenant": knownvalue.StringExact("dev-mycompany.onmicrosoft.com"),
				"client_id":    KnownValueNotEmptyString(),
				"object_id":    KnownValueNotEmptyString(),
				"auth": xknownvalue.MapExact(map[string]knownvalue.Check{
					"type":       knownvalue.StringExact("workloadIdentity"),
					"credential": knownvalue.Null(),
				}),
			}),
			"namespace_name_pattern":     knownvalue.StringExact("#{workspaceIdentifier}-#{projectIdentifier}"),
			"group_name_pattern":         knownvalue.StringExact("#{workspaceIdentifier}.#{projectIdentifier}-#{platformGroupAlias}"),
			"aks_subscription_id":        KnownValueNotEmptyString(),
			"aks_cluster_name":           knownvalue.StringExact("my-aks-cluster"),
			"aks_resource_group":         knownvalue.StringExact("my-aks-rg"),
			"send_azure_invitation_mail": knownvalue.Bool(false),
			"user_lookup_strategy":       knownvalue.StringExact("UserByMailLookupStrategy"),
			"administrative_unit_id":     knownvalue.Null(),
			"redirect_url":               knownvalue.Null(),
		}),
		"metering": knownvalue.MapExact(map[string]knownvalue.Check{
			"client_config": knownvalue.MapExact(map[string]knownvalue.Check{
				"access_token": knownvalue.MapExact(map[string]knownvalue.Check{
					"secret_value":   knownvalue.Null(),
					"secret_hash":    KnownValueNotEmptyString(),
					"secret_version": KnownValueNotEmptyString(),
				}),
			}),
			"processing": checkMeteringProcessingConfig(),
		}),
	})
}

func checkAzureRgPlatformConfig() knownvalue.Check {
	return knownvalue.MapExact(map[string]knownvalue.Check{
		"entra_tenant": knownvalue.StringExact("example-tenant.onmicrosoft.com"),
		"replication": knownvalue.MapExact(map[string]knownvalue.Check{
			"service_principal": knownvalue.MapExact(map[string]knownvalue.Check{
				"client_id": KnownValueNotEmptyString(),
				"object_id": KnownValueNotEmptyString(),
				"auth": knownvalue.MapExact(map[string]knownvalue.Check{
					"type": knownvalue.StringExact("credential"),
					"credential": knownvalue.MapExact(map[string]knownvalue.Check{
						"secret_value":   knownvalue.Null(),
						"secret_hash":    KnownValueNotEmptyString(),
						"secret_version": KnownValueNotEmptyString(),
					}),
				}),
			}),
			"subscription":                                   KnownValueNotEmptyString(),
			"resource_group_name_pattern":                    knownvalue.StringExact("#{workspaceIdentifier}-#{projectIdentifier}"),
			"user_group_name_pattern":                        knownvalue.StringExact("#{workspaceIdentifier}.#{projectIdentifier}-#{platformGroupAlias}"),
			"user_lookup_strategy":                           knownvalue.StringExact("UserByMailLookupStrategy"),
			"skip_user_group_permission_cleanup":             knownvalue.Bool(false),
			"allow_hierarchical_management_group_assignment": knownvalue.Bool(false),
			"administrative_unit_id":                         knownvalue.Null(),
			"b2b_user_invitation": knownvalue.MapExact(map[string]knownvalue.Check{
				"redirect_url":               knownvalue.StringExact("https://meshcloud.io"),
				"send_azure_invitation_mail": knownvalue.Bool(false),
			}),
			"tenant_tags": knownvalue.MapExact(map[string]knownvalue.Check{
				"namespace_prefix": knownvalue.StringExact("meshstack_"),
				"tag_mappers":      knownvalue.SetSizeExact(2),
			}),
		}),
	})
}

func checkOpenshiftPlatformConfig() knownvalue.Check {
	return knownvalue.MapExact(map[string]knownvalue.Check{
		"base_url":               knownvalue.StringExact("https://api.okd4.dev.eu-de-central.msh.host:6443"),
		"disable_ssl_validation": knownvalue.Bool(false),
		"replication": knownvalue.MapExact(map[string]knownvalue.Check{
			"client_config": knownvalue.MapExact(map[string]knownvalue.Check{
				"access_token": knownvalue.MapExact(map[string]knownvalue.Check{
					"secret_value":   knownvalue.Null(),
					"secret_hash":    KnownValueNotEmptyString(),
					"secret_version": KnownValueNotEmptyString(),
				}),
			}),
			"web_console_url":               knownvalue.StringExact("https://console-openshift-console.apps.okd4.dev.eu-de-central.msh.host"),
			"project_name_pattern":          knownvalue.StringExact("#{workspaceIdentifier}-#{projectIdentifier}"),
			"enable_template_instantiation": knownvalue.Bool(false),
			"identity_provider_name":        knownvalue.StringExact("meshStack"),
			"openshift_role_mappings": knownvalue.SetExact([]knownvalue.Check{
				knownvalue.MapExact(map[string]knownvalue.Check{
					"project_role_ref": knownvalue.MapExact(map[string]knownvalue.Check{
						"name": knownvalue.StringExact("admin"),
						"kind": knownvalue.StringExact("meshProjectRole"),
					}),
					"openshift_role": knownvalue.StringExact("admin"),
				}),
				knownvalue.MapExact(map[string]knownvalue.Check{
					"project_role_ref": knownvalue.MapExact(map[string]knownvalue.Check{
						"name": knownvalue.StringExact("user"),
						"kind": knownvalue.StringExact("meshProjectRole"),
					}),
					"openshift_role": knownvalue.StringExact("edit"),
				}),
			}),
			"tenant_tags": knownvalue.MapExact(map[string]knownvalue.Check{
				"namespace_prefix": knownvalue.StringExact("meshstack_"),
				"tag_mappers":      knownvalue.SetSizeExact(2),
			}),
		}),
		"metering": knownvalue.MapExact(map[string]knownvalue.Check{
			"client_config": knownvalue.MapExact(map[string]knownvalue.Check{
				"access_token": knownvalue.MapExact(map[string]knownvalue.Check{
					"secret_value":   knownvalue.Null(),
					"secret_hash":    KnownValueNotEmptyString(),
					"secret_version": KnownValueNotEmptyString(),
				}),
			}),
			"processing": checkMeteringProcessingConfig(),
		}),
	})
}

func checkCustomPlatformConfig() knownvalue.Check {
	return knownvalue.MapExact(map[string]knownvalue.Check{
		"platform_type_ref": knownvalue.MapExact(map[string]knownvalue.Check{
			"name": KnownValueNotEmptyString(),
			"kind": knownvalue.StringExact("meshPlatformType"),
		}),
		"metering": knownvalue.MapExact(map[string]knownvalue.Check{
			"processing": checkMeteringProcessingConfig(),
		}),
	})
}
