data "meshstack_landingzones" "for_platform" {
  # depends_on, because platform_uuid references the platform rather than the landing zone: without
  # it the query could run before the landing zone exists and list nothing.
  depends_on = [meshstack_landingzone.example]

  platform_uuid = meshstack_platform.example_custom.metadata.uuid
}
