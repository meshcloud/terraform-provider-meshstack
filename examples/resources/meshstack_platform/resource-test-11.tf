# The custom platform with the quota definitions the tenant quota cases bound their requests
# against. limits.memory only exists when var.lz_memory_default is non-zero, which is what makes it a
# landing zone default the tenant never requests.

resource "meshstack_platform" "example_custom" {
  metadata = {
    name               = "my-platform-${var.suffix}"
    owned_by_workspace = meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name      = "Example Platform"
    description       = "Custom platform using a meshPlatformType"
    endpoint          = "https://custom-platform.example.com"
    documentation_url = "https://docs.example.com"
    location_ref      = { name = "global" }

    availability = {
      restriction              = "PUBLIC"
      publication_state        = "PUBLISHED"
      restricted_to_workspaces = []
    }

    quota_definitions = concat(
      [{
        quota_key               = "limits.cpu"
        min_value               = 1
        max_value               = var.max_cpu
        unit                    = "cores"
        auto_approval_threshold = var.cpu_auto_approval_threshold
        description             = "vCPU limit"
        label                   = "CPU"
      }],
      var.lz_memory_default > 0 ? [{
        quota_key               = "limits.memory"
        min_value               = 1
        max_value               = var.lz_memory_default
        unit                    = "MiB"
        auto_approval_threshold = var.lz_memory_default
        description             = "Memory limit"
        label                   = "Memory"
      }] : [],
    )

    config = {
      custom = {
        platform_type_ref = meshstack_platform_type.example.ref

        metering = {
          processing = {
            enabled = true
          }
        }
      }
    }

    contributing_workspaces = []
  }
}
