resource "meshstack_project_tag_value" "team_a" {
  project_ref = {
    owned_by_workspace = data.meshstack_project.example.metadata.owned_by_workspace
    name               = data.meshstack_project.example.metadata.name
  }
  key   = data.meshstack_tag_definition.teams.spec.key
  value = "team-a"
}
