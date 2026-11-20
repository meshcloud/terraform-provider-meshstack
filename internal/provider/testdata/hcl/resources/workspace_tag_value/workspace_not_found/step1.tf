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

resource "meshstack_workspace_tag_value" "test" {
  metadata = {
    workspace_identifier = "does-not-exist-workspace-[[.suffix]]"
    key                  = data.meshstack_tag_definition.teams.spec.key
  }
  spec = {
    value = "team-a"
  }
}
