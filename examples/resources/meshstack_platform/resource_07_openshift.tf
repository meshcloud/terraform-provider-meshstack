resource "meshstack_platform" "example_openshift" {
  metadata = {
    name               = "my-platform"
    owned_by_workspace = "my-workspace"
  }

  spec = {
    display_name      = "Example Platform"
    description       = "OpenShift Container Platform"
    endpoint          = "https://api.okd4.dev.eu-de-central.msh.host:6443"
    documentation_url = "https://www.openshift.com"
    location_ref      = { name = "openshift" }

    availability = {
      restriction              = "PUBLIC"
      publication_state        = "PUBLISHED"
      restricted_to_workspaces = []
    }

    quota_definitions = []

    config = {
      openshift = {
        base_url               = "https://api.okd4.dev.eu-de-central.msh.host:6443"
        disable_ssl_validation = false

        replication = {
          client_config = {
            access_token = {
              plaintext = "example-openshift-service-account-token"
            }
          }

          web_console_url               = "https://console-openshift-console.apps.okd4.dev.eu-de-central.msh.host"
          project_name_pattern          = "#{workspaceIdentifier}-#{projectIdentifier}"
          enable_template_instantiation = false
          identity_provider_name        = "meshStack"

          openshift_role_mappings = [
            {
              project_role_ref = {
                name = "admin"
              }
              openshift_role = "admin"
            },
            {
              project_role_ref = {
                name = "user"
              }
              openshift_role = "edit"
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
          client_config = {
            access_token = {
              plaintext = "example-openshift-metering-token"
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
