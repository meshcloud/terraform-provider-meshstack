resource "meshstack_project_tag_value" "team_a" {
  project_ref = data.meshstack_project.example.ref
  key         = data.meshstack_tag_definition.teams.spec.key
  value       = "team-a"
}
