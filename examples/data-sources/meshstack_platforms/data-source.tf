data "meshstack_platforms" "published" {
  # All filters are optional and map to the platform list endpoint.
  publication_state = "PUBLISHED"
  # restriction        = "RESTRICTED"
  # owned_by_workspace       = "my-operator-workspace"
  # platform_type_identifier = "my-custom-platform-type"
}

# Select exactly one platform by its identifier (no hardcoded uuid needed) and reuse its
# computed `ref` as `platform_ref` in landing zone / tenant resources.
locals {
  platform = one([
    for platform in data.meshstack_platforms.published.platforms : platform
    if platform.identifier == "my-platform.global"
  ])
}
