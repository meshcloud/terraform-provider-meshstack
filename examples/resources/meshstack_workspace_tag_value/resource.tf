resource "meshstack_workspace_tag_value" "team_a" {
  metadata = {
    workspace_identifier = data.meshstack_workspace.example.metadata.name
    key                  = data.meshstack_tag_definition.teams.spec.key
  }
  spec = {
    value = "team-a"
  }
}
