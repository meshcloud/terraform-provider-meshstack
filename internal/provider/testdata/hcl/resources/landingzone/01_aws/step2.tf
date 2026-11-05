resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

data "meshstack_workspace" "example" {
  metadata = {
    name = meshstack_workspace.test.metadata.name
  }
}

resource "meshstack_platform" "test" {
  metadata = {
    name               = "test-platform-[[.suffix]]"
    owned_by_workspace = meshstack_workspace.test.metadata.name
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

          aws_identity_store = {
            identity_store_id  = "d-1234567890"
            arn                = "arn:aws:sso:::instance/ssoins-1234567890abcdef"
            group_name_pattern = "#{workspaceIdentifier}.#{projectIdentifier}-#{platformGroupAlias}"
            sign_in_url        = "https://d-1234567890.awsapps.com/start"

            aws_role_mappings = [
              {
                project_role_ref    = { name = "admin" }
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

data "meshstack_platform" "example" {
  metadata = {
    uuid = meshstack_platform.test.metadata.uuid
  }
}

resource "meshstack_tag_definition" "confidentiality" {
  spec = {
    target_kind  = "meshLandingZone"
    key          = "confidentiality-[[.suffix]]"
    display_name = "Confidentiality"
    value_type = {
      string = {}
    }
  }
}

resource "meshstack_tag_definition" "environment" {
  spec = {
    target_kind  = "meshLandingZone"
    key          = "environment-[[.suffix]]"
    display_name = "Environment"
    value_type = {
      multi_select = {
        options = ["dev", "qa", "test"]
      }
    }
  }
}

resource "meshstack_landingzone" "example" {
  metadata = {
    name               = "test-lz-[[.suffix]]"
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
    tags = {
      "confidentiality-[[.suffix]]" = ["internal"],
      "environment-[[.suffix]]"     = ["dev", "qa", "test"],
    }
  }
  spec = {
    display_name                  = "My Landing Zone's Display Name"
    description                   = "My Landing Zone Description"
    automate_deletion_approval    = false
    automate_deletion_replication = false
    restricted                    = false
    info_link                     = "https://example.com/info-about-aws-landing-zone"
    platform_ref                  = data.meshstack_platform.example.ref
    platform_properties = {
      aws = {
        aws_target_org_unit_id = "ou-lpzq-kmf17bec"
        aws_enroll_account     = true
        aws_lambda_arn         = "arn:aws:lambda:us-east-1:123456789012:function:MyLambdaFunction"
        aws_role_mappings = [
          {
            project_role_ref = { name = "reader" }
            platform_role    = "project-reader"
            policies = [
              "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
            ]
          }
        ]
      }
    }
  }
}
