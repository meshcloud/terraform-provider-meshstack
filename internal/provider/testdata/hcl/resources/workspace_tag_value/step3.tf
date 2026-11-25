resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

resource "meshstack_tag_definition" "teams" {
  spec = {
    target_kind  = "meshWorkspace"
    key          = "teams-[[.suffix]]"
    display_name = "Teams"
    value_type = {
      multi_select = {
        options = ["team-a", "team-b", "team-c"]
      }
    }
  }
}

data "meshstack_tag_definition" "teams" {
  metadata = {
    name = meshstack_tag_definition.teams.metadata.name
  }
}

data "meshstack_workspace" "example" {
  metadata = {
    name = meshstack_workspace.test.metadata.name
  }
}

[[ example "resources/meshstack_workspace_tag_value/resource.tf" ]]

resource "meshstack_workspace_tag_value" "team_b" {
  workspace = data.meshstack_workspace.example.metadata.name
  key       = data.meshstack_tag_definition.teams.spec.key
  value     = "team-b"
}
