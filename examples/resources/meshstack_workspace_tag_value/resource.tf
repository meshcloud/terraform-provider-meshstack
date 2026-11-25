resource "meshstack_workspace_tag_value" "team_a" {
  workspace = data.meshstack_workspace.example.metadata.name
  key       = data.meshstack_tag_definition.teams.spec.key
  value     = "team-a"
}
