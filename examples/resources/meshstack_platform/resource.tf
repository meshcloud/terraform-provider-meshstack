resource "meshstack_platform" "example" {
  metadata = {
    name               = "my-azure-platform"
    owned_by_workspace = "my-workspace"
  }

  spec = {
    display_name      = "Azure"
    description       = "Azure is the Public Cloud Service provided by Microsoft."
    endpoint          = "https://azure.microsoft.com"
    documentation_url = "https://azure.microsoft.com"

    location_ref = {
      name = "meshcloud-azure-dev"
    }

    availability = {
      restriction              = "PUBLIC"
      publication_state        = "PUBLISHED"
      restricted_to_workspaces = []
    }

    quota_definitions = []

    config = {
      azure = {
        entra_tenant = "dev-mycompany.onmicrosoft.com"

        replication = {
          service_principal = {
            client_id = "58d6f907-7b0e-4fd8-b328-3e8342dddc8d"
            object_id = "3c305efe-625d-4eaf-9bfa-b981ddbcc99f"
            # Workload Identity Federation (Recommended)
            # To use workload identity federation, set auth to an empty object
            auth = {}

            # Credential-based authentication (Alternative)
            # Uncomment the following to use client secret authentication instead
            # auth = {
            #   credential = {
            #     plaintext = "your-client-secret-here"
            #   }
            # }
          }

          provisioning = {
            subscription_owner_object_ids = [
              "2af5651f-bfa2-45b8-8780-f63dd51f515f"
            ]

            pre_provisioned = {
              unused_subscription_name_prefix = "unused-"
            }
          }

          b2b_user_invitation = {
            redirect_url               = "https://portal.azure.com/#home"
            send_azure_invitation_mail = false
          }

          subscription_name_pattern   = "#{workspaceIdentifier}.#{projectIdentifier}"
          group_name_pattern          = "#{workspaceIdentifier}.#{projectIdentifier}-#{platformGroupAlias}"
          blueprint_service_principal = "ce0c3688-3247-4083-b49f-33fdbac1ea65"
          blueprint_location          = "westeurope"

          azure_role_mappings = [
            {
              project_role_ref = {
                name = "admin"
              }
              azure_role = {
                alias = "admin"
                id    = "b69d42fd-1e97-47d0-958d-3ce50d18af71"
              }
            },
            {
              project_role_ref = {
                name = "reader"
              }
              azure_role = {
                alias = "reader"
                id    = "9c4cbbde-f2da-479e-9709-0f9ca8fa69df"
              }
            },
            {
              project_role_ref = {
                name = "user"
              }
              azure_role = {
                alias = "user"
                id    = "7eeffa89-84ca-4106-9677-c9206b2fc14d"
              }
            }
          ]

          tenant_tags = {
            namespace_prefix = "meshstack_"

            tag_mappers = [
              {
                key           = "wident"
                value_pattern = "$${workspaceIdentifier}"
              },
              {
                key           = "pident"
                value_pattern = "prefix-$${projectIdentifier}"
              },
              {
                key           = "pname"
                value_pattern = "$${projectName}"
              },
              {
                key           = "wname"
                value_pattern = "$${workspaceName}"
              },
              {
                key           = "paymentIdentifier"
                value_pattern = "$${paymentIdentifier}"
              },
              {
                key           = "paymentName"
                value_pattern = "$${paymentName}"
              },
              {
                key           = "paymentExpirationDate"
                value_pattern = "$${paymentExpirationDate}"
              }
            ]
          }

          user_lookup_strategy                           = "UserByMailLookupStrategy"
          skip_user_group_permission_cleanup             = false
          allow_hierarchical_management_group_assignment = false
        }
      }
    }

    contributing_workspaces = []
  }
}
