resource "meshstack_building_block_definition" "test" {
  metadata = { owned_by_workspace = "test-ws-[[.suffix]]" }
  spec     = { display_name = "Test", description = "Test" }
  version_spec = {
    draft          = true
    inputs         = { candidate = { display_name = "Candidate", type = "STRING", assignment_type = "STATIC", argument = jsonencode("c"), is_optional = true } }
    implementation = { terraform = { terraform_version = "1.9.0", repository_url = "https://github.com/example/bb.git" } }
  }
}
