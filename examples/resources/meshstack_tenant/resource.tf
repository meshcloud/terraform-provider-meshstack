resource "meshstack_tenant" "example" {
  metadata = {
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
    owned_by_project   = data.meshstack_project.example.metadata.name
  }

  spec = {
    platform_ref     = data.meshstack_platform.example.ref
    landing_zone_ref = data.meshstack_landingzone.example.ref
  }

  # wait until the tenant's platform_tenant_id is set (not necessarily full replication); defaults to true
  wait_for_completion = true
}
