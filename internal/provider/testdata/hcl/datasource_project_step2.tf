resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-{{.suffix}}"
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

resource "meshstack_project" "test" {
  metadata = {
    name               = "my-project"
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }
  spec = {
    display_name = "My Project's Display Name"
  }
}

{{ example "data-sources/meshstack_project/data-source.tf" }}
