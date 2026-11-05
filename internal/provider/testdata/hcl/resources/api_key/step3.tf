resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

data "meshstack_workspace" "example" {
  metadata = {
    name = meshstack_workspace.test.metadata.name
  }
}

resource "meshstack_api_key" "example" {
  metadata = {
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name = "ci-key"
    permissions  = ["LANDINGZONE_LIST", "PROJECT_LIST"]
    expires_at   = "2099-06-30"
  }
}
