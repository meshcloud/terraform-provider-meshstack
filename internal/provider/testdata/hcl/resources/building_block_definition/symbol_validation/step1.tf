resource "meshstack_building_block_definition" "test" {
  metadata = { owned_by_workspace = "test-ws-[[.suffix]]" }
  spec = {
    display_name = "Test"
    description  = "Test"
    symbol       = "[[.symbol]]"
  }
  version_spec = {
    draft          = true
    implementation = { manual = {} }
  }
}
