resource "meshstack_building_block_definition" "test" {
  metadata = { owned_by_workspace = "test-ws-[[.suffix]]" }
  spec = {
    display_name = "Test"
    description  = "Test"
    target_type  = "WORKSPACE_LEVEL"
  }
  version_spec = {
    draft          = true
    implementation = { terraform = { terraform_version = "1.9.0", repository_url = "https://example.com/bb.git" } }
    inputs = {
      cost_center = {
        display_name    = "Cost Center"
        assignment_type = "TAG"
        type            = "CODE"
        argument        = jsonencode("WORKSPACE.costCenter")
      }
    }
  }
}
