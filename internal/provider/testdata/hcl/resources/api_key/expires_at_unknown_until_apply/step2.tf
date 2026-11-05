resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

resource "terraform_data" "expiry" {
  input = "2099-06-30"
}

resource "meshstack_api_key" "test" {
  metadata = {
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }

  spec = {
    display_name = "ci-key"
    permissions  = ["LANDINGZONE_LIST", "PROJECT_LIST"]
    expires_at   = terraform_data.expiry.output
  }
}
