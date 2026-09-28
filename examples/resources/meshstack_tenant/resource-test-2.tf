# The tenant's refs resolved through data sources rather than the resources directly: the platform
# from the plural list (a one(...) select) and the landing zone from the singular one, so both
# cardinalities are exercised. The assertions then compare the other-cardinality data source against
# these, proving the plural element `ref` and the singular `ref` are interchangeable.

data "meshstack_platform" "example" {
  metadata = {
    uuid = meshstack_platform.example_custom.metadata.uuid
  }
}

data "meshstack_landingzone" "example" {
  metadata = {
    name = meshstack_landingzone.example.metadata.name
  }
}

# The plural data sources filter by workspace / platform, which does not depend on the object rows
# themselves, so they need an explicit depends_on to list after those rows exist.
data "meshstack_platforms" "published" {
  depends_on = [meshstack_platform.example_custom]

  owned_by_workspace = meshstack_workspace.example.metadata.name
}

data "meshstack_landingzones" "for_platform" {
  depends_on = [meshstack_landingzone.example]

  platform_uuid = meshstack_platform.example_custom.metadata.uuid
}

resource "meshstack_tenant" "example" {
  metadata = {
    owned_by_workspace = meshstack_project.example.metadata.owned_by_workspace
    owned_by_project   = meshstack_project.example.metadata.name
  }

  spec = {
    platform_ref = one([
      for p in data.meshstack_platforms.published.platforms : p
      if p.metadata.uuid == meshstack_platform.example_custom.metadata.uuid
    ]).ref
    landing_zone_ref = data.meshstack_landingzone.example.ref
  }

  wait_for_completion = true
}
