resource "meshstack_platform" "example_gcp" {
  metadata = {
    name               = "my-platform"
    owned_by_workspace = "my-workspace"
  }

  spec = {
    display_name      = "Example Platform"
    description       = "Google Cloud Platform"
    endpoint          = "https://console.cloud.google.com"
    documentation_url = "https://cloud.google.com"

    location_ref = { name = "gcp-meshstack-dev" }

    availability = {
      restriction              = "PUBLIC"
      publication_state        = "PUBLISHED"
      restricted_to_workspaces = []
    }

    quota_definitions = []

    config = {
      gcp = {
        replication = {
          service_account = {
            credential = {
              secret_value = "top-secret-ephemeral"
            }
          }

          project_id_pattern                   = "#{workspaceIdentifier}-#{projectIdentifier}"
          project_name_pattern                 = "#{workspaceIdentifier}.#{projectIdentifier}"
          group_name_pattern                   = "#{workspaceIdentifier}.#{projectIdentifier}-#{platformGroupAlias}"
          billing_account_id                   = "012345-6789AB-CDEF01"
          domain                               = "example.com"
          customer_id                          = "C01234567"
          user_lookup_strategy                 = "email"
          allow_hierarchical_folder_assignment = false
          skip_user_group_permission_cleanup   = false

          gcp_role_mappings = [
            {
              project_role_ref = {
                name = "admin"
              }
              gcp_role = "roles/editor"
            },
            {
              project_role_ref = {
                name = "reader"
              }
              gcp_role = "roles/viewer"
            }
          ]

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
          service_account = {
            credential = {
              secret_value = "top-secret-ephemeral"
            }
          }

          dataset_id            = "cloud_costs"
          bigquery_table        = "gcp_billing_export_v1"
          partition_time_column = "usage_start_time"

          processing = {
            enabled = true
          }
        }
      }
    }

    contributing_workspaces = []
  }
}
