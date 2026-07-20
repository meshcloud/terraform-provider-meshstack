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
