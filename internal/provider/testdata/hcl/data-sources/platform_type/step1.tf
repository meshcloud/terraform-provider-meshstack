resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

resource "meshstack_platform_type" "test" {
  metadata = {
    name               = "TEST-PT-[[.upper_suffix]]"
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }

  spec = {
    display_name     = "My Custom Platform [[.suffix]]"
    default_endpoint = "https://platform.example.com"
    icon             = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciLz4="
  }
}
