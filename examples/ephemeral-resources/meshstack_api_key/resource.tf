ephemeral "meshstack_api_key" "example" {
  workspace_identifier = "my-workspace"
  name                 = "temporary-api-key"
  authorities = [
    "workspace.read",
    "project.read"
  ]
  expiry_date = "2026-01-01T00:00:00Z"
}
