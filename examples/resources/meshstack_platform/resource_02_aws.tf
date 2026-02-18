resource "meshstack_platform" "example_aws" {
  metadata = {
    name               = "my-platform"
    owned_by_workspace = "my-workspace"
  }

  spec = {
    display_name      = "Example Platform"
    description       = "Amazon Web Services"
    endpoint          = "https://console.aws.amazon.com"
    documentation_url = "https://aws.amazon.com"

    location_ref = { name = "aws-meshstack-dev" }

    availability = {
      restriction              = "PUBLIC"
      publication_state        = "PUBLISHED"
      restricted_to_workspaces = []
    }

    quota_definitions = []

    config = {
      aws = {
        region = "us-east-1"

        replication = {
          access_config = {
            organization_root_account_role = "OrganizationAccountAccessRole"
            auth = {
              credential = {
                access_key = "AKIAIOSFODNN7EXAMPLE"
                secret_key = {
                  secret_value = "top-secret-ephemeral"
                }
              }
            }
          }

          account_alias_pattern                             = "#{workspaceIdentifier}-#{projectIdentifier}"
          account_email_pattern                             = "aws+#{workspaceIdentifier}.#{projectIdentifier}@example.com"
          automation_account_role                           = "OrganizationAccountAccessRole"
          account_access_role                               = "OrganizationAccountAccessRole"
          self_downgrade_access_role                        = false
          enforce_account_alias                             = false
          wait_for_external_avm                             = false
          skip_user_group_permission_cleanup                = false
          allow_hierarchical_organizational_unit_assignment = false

          aws_sso = {
            arn                = "arn:aws:sso:::instance/ssoins-1234567890abcdef"
            scim_endpoint      = "https://scim.us-east-1.amazonaws.com/abcd1234-5678-90ab-cdef-example12345/scim/v2/"
            group_name_pattern = "#{workspaceIdentifier}.#{projectIdentifier}-#{platformGroupAlias}"
            sso_access_token = {
              secret_value = "top-secret-ephemeral"
            }
            sign_in_url = "https://my-sso-portal.awsapps.com/start"

            aws_role_mappings = [
              {
                project_role_ref = {
                  name = "admin"
                }
                aws_role            = "admin"
                permission_set_arns = ["arn:aws:sso:::permissionSet/ssoins-1234567890abcdef/ps-1234567890abcdef"]
              }
            ]
          }

          tenant_tags = {
            namespace_prefix = "meshstack_"

            tag_mappers = [
              {
                key           = "workspace"
                value_pattern = "$${workspaceIdentifier}"
              },
              {
                key           = "project"
                value_pattern = "$${projectIdentifier}"
              }
            ]
          }
        }

        metering = {
          access_config = {
            organization_root_account_role = "OrganizationAccountAccessRole"
            auth = {
              credential = {
                access_key = "AKIAIOSFODNN7EXAMPLE"
                secret_key = {
                  secret_value = "top-secret-ephemeral"
                }
              }
            }
          }

          filter                            = "NONE"
          reserved_instance_fair_chargeback = false
          savings_plan_fair_chargeback      = false

          processing = {
            enabled = true
          }
        }
      }
    }

    contributing_workspaces = []
  }
}
