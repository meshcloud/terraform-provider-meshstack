resource "meshstack_building_block_definition" "test" {
  metadata = { owned_by_workspace = "test-ws-[[.suffix]]" }
  spec = {
    display_name      = "Test"
    description       = "Test"
    approval_policies = {}
    schedule          = { mode = "DRIFT_DETECTION", frequency = "DAILY", automatic_approval = true }
  }
  version_spec = {
    draft          = true
    implementation = { terraform = { terraform_version = "1.9.0", repository_url = "https://example.com/bb.git" } }
  }
}
