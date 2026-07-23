resource "meshstack_platform" "example_aks" {
  metadata = {
    name               = "my-platform"
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name      = "Example Platform"
    description       = "Azure Kubernetes Service"
    endpoint          = "https://myaks-dns.westeurope.azmk8s.io:443"
    documentation_url = "https://azure.microsoft.com/en-us/services/kubernetes-service"

    location_ref = { name = "eu-de-central" }

    availability = {
      restriction              = "PUBLIC"
      publication_state        = "PUBLISHED"
      restricted_to_workspaces = []
    }

    quota_definitions = []

    config = {
      aks = {
        base_url               = "https://myaks-dns.westeurope.azmk8s.io:443"
        disable_ssl_validation = false

        replication = {
          # non_ephemeral_secret builds the secret block for a value kept in config or state. It sets
          # secret_version to sha256(secret_value), so the value is sent again whenever it changes.
          # For an ephemeral secret, set secret_value from an ephemeral resource and manage
          # secret_version yourself.
          access_token = provider::meshstack::non_ephemeral_secret("top-secret-value")

          service_principal = {
            entra_tenant = "dev-mycompany.onmicrosoft.com"
            client_id    = "58d6f907-7b0e-4fd8-b328-3e8342dddc8d"
            object_id    = "3c305efe-625d-4eaf-9bfa-b981ddbcc99f"
            # Workload Identity Federation (Recommended)
            auth = {}

            # Credential-based authentication (Alternative)
            # auth = {
            #   credential = {
            #     secret_value = "top-secret-ephemeral"
            #   }
            # }
          }

          namespace_name_pattern     = "#{workspaceIdentifier}-#{projectIdentifier}"
          group_name_pattern         = "#{workspaceIdentifier}.#{projectIdentifier}-#{platformGroupAlias}"
          aks_subscription_id        = "12345678-90ab-cdef-1234-567890abcdef"
          aks_cluster_name           = "my-aks-cluster"
          aks_resource_group         = "my-aks-rg"
          send_azure_invitation_mail = false
          user_lookup_strategy       = "UserByMailLookupStrategy"
        }

        metering = {
          client_config = {
            access_token = {
              secret_value = "top-secret-ephemeral"
            }
          }

          processing = {
            enabled = true
          }
        }
      }
    }

    contributing_workspaces = []
  }
}
