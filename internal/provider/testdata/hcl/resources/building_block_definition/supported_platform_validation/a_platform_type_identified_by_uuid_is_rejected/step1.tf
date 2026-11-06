resource "meshstack_building_block_definition" "test" {
  metadata = { owned_by_workspace = "test-ws-[[.suffix]]" }
  spec = {
    display_name        = "Test"
    description         = "Test"
    target_type         = "TENANT_LEVEL"
    supported_platforms = [{ kind = "meshPlatformType", uuid = "3f1c0f2e-1a2b-4c3d-8e9f-0a1b2c3d4e5f" }]
  }
  version_spec = {
    draft          = true
    implementation = { manual = {} }
  }
}
