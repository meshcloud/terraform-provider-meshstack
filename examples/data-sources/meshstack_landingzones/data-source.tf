data "meshstack_landingzones" "for_platform" {
  # List the landing zones of a chosen platform (resolve the uuid via meshstack_platforms).
  platform_uuid = one(data.meshstack_platforms.published.platforms).metadata.uuid
  restricted    = false
  # identifier         = "my-landing-zone"
  # owned_by_workspace = "my-operator-workspace"
}

# Select one landing zone and reuse its computed `ref` as `landing_zone_ref` in tenant resources.
locals {
  landing_zone = one([
    for landing_zone in data.meshstack_landingzones.for_platform.landing_zones : landing_zone
    if !landing_zone.status.disabled
  ])
}
