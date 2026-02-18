resource "meshstack_platform" "example_azurerg" {
  metadata = {
    name               = "my-platform"
    owned_by_workspace = "my-workspace"
  }

  spec = {
    display_name      = "Example Platform"
    description       = "Azure Resource Group platform for tenant isolation"
    endpoint          = "https://portal.azure.com"
    documentation_url = "https://azure.microsoft.com"
    location_ref      = { name = "meshcloud-azure-dev" }

    availability = {
      restriction              = "PUBLIC"
      publication_state        = "PUBLISHED"
      restricted_to_workspaces = []
    }

    quota_definitions = []

    config = {
      azurerg = {
        entra_tenant = "example-tenant.onmicrosoft.com"

        replication = {
          service_principal = {
            client_id = "12345678-1234-1234-1234-123456789abc"
            object_id = "87654321-4321-4321-4321-cba987654321"
            auth = {
              credential = {
                secret_value = "top-secret-ephemeral"
              }
            }
          }

          subscription                                   = "12345678-1234-1234-1234-123456789abc"
          resource_group_name_pattern                    = "#{workspaceIdentifier}-#{projectIdentifier}"
          user_group_name_pattern                        = "#{workspaceIdentifier}.#{projectIdentifier}-#{platformGroupAlias}"
          user_lookup_strategy                           = "UserByMailLookupStrategy"
          skip_user_group_permission_cleanup             = false
          allow_hierarchical_management_group_assignment = false

          b2b_user_invitation = {
            redirect_url               = "https://meshcloud.io"
            send_azure_invitation_mail = false
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
      }
    }

    contributing_workspaces = []
  }
}
