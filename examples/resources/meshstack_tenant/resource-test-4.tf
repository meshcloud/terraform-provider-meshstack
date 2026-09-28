resource "meshstack_tenant" "example" {
  metadata = {
    owned_by_workspace = meshstack_project.example.metadata.owned_by_workspace
    owned_by_project   = meshstack_project.example.metadata.name
  }

  spec = {
    # A synthetic uuid: the plan action is decided before any backend validation, so this platform
    # never has to exist for RequiresReplace to be observable.
    platform_ref     = { kind = "meshPlatform", uuid = "11111111-1111-1111-1111-111111111111" }
    landing_zone_ref = meshstack_landingzone.example.ref
  }

  wait_for_completion = true
}
