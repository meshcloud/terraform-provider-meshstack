resource "meshstack_tenant" "example" {
  metadata = {
    owned_by_workspace = meshstack_project.example.metadata.owned_by_workspace
    owned_by_project   = meshstack_project.example.metadata.name
  }

  spec = {
    platform_ref     = meshstack_platform.example_custom.ref
    landing_zone_ref = meshstack_landingzone.example.ref

    # spec.requested_quotas is create-only and Optional: it echoes the config verbatim, while the
    # effective quotas come back in status.applied_quotas.
    requested_quotas = {
      "limits.cpu" = { value = var.requested_cpu }
    }
  }

  wait_for_completion = true
}
