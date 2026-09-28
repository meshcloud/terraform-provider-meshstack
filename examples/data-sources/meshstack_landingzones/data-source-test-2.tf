# The listing the consumer workspace runs, through the meshstack-other provider alias backed by its
# restricted API key.

data "meshstack_landingzones" "for_platform" {
  provider = meshstack-other

  platform_uuid = meshstack_platform.example_custom.metadata.uuid
}
