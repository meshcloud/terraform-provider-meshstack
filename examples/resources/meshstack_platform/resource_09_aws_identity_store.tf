resource "meshstack_platform" "example_aws_identity_store" {
  metadata = {
    name               = "my-platform"
    owned_by_workspace = "my-workspace"
  }

  spec = {
    display_name      = "Example Platform"
    description       = "Amazon Web Services (Identity Store)"
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
              workload_identity = {
                role_arn = "arn:aws:iam::123456789:role/MeshfedServiceRole"
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

          aws_identity_store = {
            identity_store_id  = "d-1234567890"
            arn                = "arn:aws:sso:::instance/ssoins-123456789abc"
            group_name_pattern = "#{workspaceIdentifier}.#{projectIdentifier}-#{platformGroupAlias}"
            sign_in_url        = "https://d-1234567890.awsapps.com/start"

            aws_role_mappings = [
              {
                project_role_ref = {
                  name = "admin"
                }
                aws_role            = "admin"
                permission_set_arns = ["arn:aws:sso:::permissionSet/ssoins-123456789abc/ps-abc123"]
              }
            ]
          }
        }
      }
    }

    contributing_workspaces = []
  }
}
