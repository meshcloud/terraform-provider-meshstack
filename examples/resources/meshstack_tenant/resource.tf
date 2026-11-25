resource "meshstack_tenant" "example" {
  metadata = {
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
    owned_by_project   = data.meshstack_project.example.metadata.name
  }

  spec = {
    platform_ref     = local.platform.ref
    landing_zone_ref = local.landing_zone.ref
  }

  # wait until the tenant's platform_tenant_id is set (not necessarily full replication); defaults to true
  wait_for_completion = true

  # on destroy, wait until meshStack has deleted the tenant; defaults to wait_for_completion
  wait_for_deletion = true

  # how long the waits above may take; both default to 30m
  timeouts = {
    create = "30m"
    delete = "30m"
  }

  # The provider only warns when the creation does not complete in time; this fails the apply without tainting the tenant.
  lifecycle {
    postcondition {
      condition     = self.spec.platform_tenant_id != null
      error_message = "Tenant ${self.metadata.uuid} has no platform tenant ID yet. Check its replication in meshPanel."
    }
  }
}

# Resolve the platform and landing zone from the plural (marketplace) data sources instead of
# hardcoding uuids. one(...) selects exactly one element, whose computed `ref` feeds the tenant above.
data "meshstack_platforms" "available" {
  owned_by_workspace = data.meshstack_workspace.example.metadata.name
}

data "meshstack_landingzones" "available" {
  platform_uuid = local.platform.metadata.uuid
}

locals {
  platform = one([
    for p in data.meshstack_platforms.available.platforms : p if p.identifier == "my-platform.global"
  ])
  landing_zone = one([
    for lz in data.meshstack_landingzones.available.landing_zones : lz if lz.metadata.name == "my-landing-zone"
  ])
}
