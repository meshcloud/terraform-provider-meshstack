resource "meshstack_project_tag_value" "team_a" {
  metadata = {
    workspace_identifier = data.meshstack_project.example.metadata.owned_by_workspace
    project_identifier   = data.meshstack_project.example.metadata.name
    key                  = data.meshstack_tag_definition.teams.spec.key
  }
  spec = {
    value = "team-a"
  }
}
