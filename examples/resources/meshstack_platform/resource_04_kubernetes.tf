resource "meshstack_platform" "example_kubernetes" {
  metadata = {
    name               = "my-platform"
    owned_by_workspace = "my-workspace"
  }

  spec = {
    display_name      = "Example Platform"
    description       = "Kubernetes Cluster"
    endpoint          = "https://k8s.dev.eu-de-central.msh.host:6443"
    documentation_url = "https://kubernetes.io"

    location_ref = { name = "global" }

    availability = {
      restriction              = "PUBLIC"
      publication_state        = "PUBLISHED"
      restricted_to_workspaces = []
    }

    quota_definitions = []

    config = {
      kubernetes = {
        base_url               = "https://k8s.dev.eu-de-central.msh.host:6443"
        disable_ssl_validation = false

        replication = {
          client_config = {
            access_token = {
              plaintext = "mock-k8s-access-token"
            }
          }

          namespace_name_pattern = "#{workspaceIdentifier}-#{projectIdentifier}"
        }

        metering = {
          client_config = {
            access_token = {
              plaintext = "mock-k8s-metering-token"
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
