# meshPlatform resource example

terraform {
  required_providers {
    meshstack = {
      source = "meshcloud/meshstack"
    }
  }
}

provider "meshstack" {
  # Configuration via environment variables:
  # MESHSTACK_ENDPOINT
  # MESHSTACK_API_KEY
  # MESHSTACK_API_SECRET
}

# AWS meshPlatform resource example
resource "meshstack_platform" "aws_example" {
  metadata = {
    name               = "my-aws-platform"
    owned_by_workspace = "example-workspace"
  }

  spec = {
    display_name = "Production AWS Platform"
    location_ref = {
      identifier = "aws-us-west-2"
    }
    description = "Production AWS platform for compute workloads"
    endpoint    = "https://console.aws.amazon.com/"

    availability = {
      restriction        = "PUBLIC"
      marketplace_status = "PUBLISHED"
    }

    contributing_workspaces = ["platform-team"]

    config = {
      type = "aws"
      aws = {
        region = "us-west-2"

        replication = {
          access_config = {
            organization_root_account_role = "arn:aws:iam::123456789012:role/MeshfedServiceRole"

            # Option 1: Service User Authentication (using access keys)
            service_user_config = {
              access_key = "AKIAIOSFODNN7EXAMPLE"
              secret_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
            }

            # Option 2: Workload Identity Authentication (recommended)
            # workload_identity_config = {
            #   role_arn = "arn:aws:iam::123456789012:role/MeshfedWorkloadIdentityRole"
            # }
          }

          wait_for_external_avm              = false
          automation_account_role            = "arn:aws:iam::987654321098:role/MeshfedAutomationRole"
          account_access_role                = "MeshAccountAccessRole"
          account_alias_pattern              = "#{workspaceIdentifier}-#{projectIdentifier}"
          enforce_account_alias              = true
          account_email_pattern              = "aws+#{workspaceIdentifier}.#{projectIdentifier}@company.com"
          self_downgrade_access_role         = true
          skip_user_group_permission_cleanup = false

          # Optional: Tenant Tags Configuration
          tenant_tags = {
            namespace_prefix = "meshstack_"
            tag_mappers = [
              {
                key           = "workspace"
                value_pattern = "#{workspaceIdentifier}"
              },
              {
                key           = "project"
                value_pattern = "#{projectIdentifier}"
              }
            ]
          }

          # Optional: AWS SSO Configuration
          # aws_sso = {
          #   scim_endpoint        = "https://scim.us-east-1.amazonaws.com/12345678-1234-1234-1234-123456789abc/scim/v2/"
          #   arn                  = "arn:aws:sso:::instance/ssoins-123456789abc"
          #   group_name_pattern   = "#{workspaceIdentifier}-#{projectIdentifier}-#{platformGroupAlias}"
          #   sso_access_token     = "example-access-token-here"
          #   sign_in_url          = "https://my-company.awsapps.com/start"
          #   role_mappings = {
          #     admin = {
          #       aws_role_name       = "Administrator"
          #       permission_set_arns = ["arn:aws:sso:::permissionSet/ssoins-123456789abc/ps-123456789abcdef0"]
          #     }
          #     member = {
          #       aws_role_name       = "Developer"
          #       permission_set_arns = ["arn:aws:sso:::permissionSet/ssoins-123456789abc/ps-fedcba9876543210"]
          #     }
          #   }
          # }

          # Optional: Enrollment Configuration (for AWS Control Tower)
          # enrollment_configuration = {
          #   management_account_id        = "123456789012"
          #   account_factory_product_id   = "prod-1234567890abcdef0"
          # }
        }
      }
    }
  }
}

# Azure meshPlatform resource example
resource "meshstack_platform" "azure_example" {
  metadata = {
    name               = "my-azure-platform"
    owned_by_workspace = "example-workspace"
  }

  spec = {
    display_name = "Production Azure Platform"
    location_ref = {
      identifier = "azure-westeurope"
    }
    description = "Production Azure platform for enterprise workloads"
    endpoint    = "https://portal.azure.com/"

    availability = {
      restriction        = "PRIVATE"
      marketplace_status = "PUBLISHED"
    }

    contributing_workspaces = ["platform-team"]

    config = {
      type = "azure"
      azure = {
        entra_tenant = "12345678-1234-1234-1234-123456789abc"

        replication = {
          service_principal = {
            client_id                      = "87654321-4321-4321-4321-cba987654321"
            auth_type                      = "CREDENTIALS"
            credentials_auth_client_secret = "super-secret-client-secret"
            object_id                      = "11111111-1111-1111-1111-111111111111"
          }

          subscription_name_pattern          = "#{workspaceIdentifier}-#{projectIdentifier}"
          group_name_pattern                 = "#{workspaceIdentifier}-#{projectIdentifier}-#{platformGroupAlias}"
          blueprint_service_principal        = "22222222-2222-2222-2222-222222222222"
          blueprint_location                 = "West Europe"
          user_look_up_strategy              = "email"
          skip_user_group_permission_cleanup = false

          role_mappings = {
            admin = {
              alias = "Administrator"
              id    = "8e3af657-a8ff-443c-a75c-2fe8c4bcb635"
            }
            member = {
              alias = "Contributor"
              id    = "b24988ac-6180-42a0-ab88-20f7382dd24c"
            }
          }

          # Optional: Subscription provisioning via Enterprise Enrollment
          # provisioning = {
          #   enterprise_enrollment = {
          #     enrollment_account_id                   = "123456"
          #     subscription_offer_type                 = "MS-AZR-0017P"
          #     use_legacy_subscription_enrollment      = false
          #     subscription_creation_error_cooldown_sec = 900
          #   }
          # }

          # Optional: B2B User Invitation
          # b2b_user_invitation = {
          #   redirect_url               = "https://portal.azure.com"
          #   send_azure_invitation_mail = true
          # }
        }
      }
    }
  }
}

# GCP meshPlatform resource example
resource "meshstack_platform" "gcp_example" {
  metadata = {
    name               = "my-gcp-platform"
    owned_by_workspace = "example-workspace"
  }

  spec = {
    display_name = "Production GCP Platform"
    location_ref = {
      identifier = "gcp-us-central1"
    }
    description = "Production Google Cloud Platform for modern workloads"
    endpoint    = "https://console.cloud.google.com/"

    availability = {
      restriction              = "RESTRICTED"
      restricted_to_workspaces = ["approved-workspace-1", "approved-workspace-2"]
      marketplace_status       = "REQUESTED"
    }

    contributing_workspaces = ["platform-team", "cloud-engineering"]

    config = {
      type = "gcp"
      gcp = {
        replication = {
          service_account_config = {
            # Option 1: Service Account Credentials
            service_account_credentials_config = {
              service_account_credentials_b64 = "ewogICJ0eXBlIjogInNlcnZpY2VfYWNjb3VudCIsCiAgInByb2plY3RfaWQiOiAibXktcHJvamVjdCIKfQ=="
            }

            # Option 2: Workload Identity (recommended)
            # service_account_workload_identity_config = {
            #   audience              = "//iam.googleapis.com/projects/123456789/locations/global/workloadIdentityPools/my-pool/providers/my-provider"
            #   service_account_email = "meshstack-replicator@my-project.iam.gserviceaccount.com"
            # }
          }

          domain                               = "company.com"
          customer_id                          = "C01234567"
          group_name_pattern                   = "#{workspaceIdentifier}-#{projectIdentifier}-#{platformGroupAlias}"
          project_name_pattern                 = "#{workspaceIdentifier}-#{projectIdentifier}"
          project_id_pattern                   = "#{workspaceIdentifier}-#{projectIdentifier}"
          billing_account_id                   = "012345-6789AB-CDEFGH"
          user_lookup_strategy                 = "email"
          allow_hierarchical_folder_assignment = true
          skip_user_group_permission_cleanup   = false

          role_mappings = {
            "admin"  = "roles/owner"
            "member" = "roles/editor"
            "viewer" = "roles/viewer"
          }
        }
      }
    }
  }
}

output "aws_platform_id" {
  description = "The AWS platform identifier"
  value       = meshstack_platform.aws_example.metadata.name
}

output "azure_platform_id" {
  description = "The Azure platform identifier"
  value       = meshstack_platform.azure_example.metadata.name
}

output "gcp_platform_id" {
  description = "The GCP platform identifier"
  value       = meshstack_platform.gcp_example.metadata.name
}
