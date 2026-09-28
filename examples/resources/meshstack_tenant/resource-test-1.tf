resource "meshstack_tenant" "example" {
  metadata = {
    owned_by_workspace = meshstack_project.example.metadata.owned_by_workspace
    owned_by_project   = meshstack_project.example.metadata.name
  }

  spec = {
    platform_ref     = meshstack_platform.example_custom.ref
    landing_zone_ref = meshstack_landingzone.example.ref
  }

  # wait until the tenant's platform_tenant_id is set (not necessarily full replication); defaults to true
  wait_for_completion = true
}
