resource "meshstack_integration" "example_gitlab" {
  metadata = {
    owned_by_workspace = meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name = "GitLab Integration"
    config = {
      gitlab = {
        base_url = "https://gitlab.com"
      }
    }
  }
}
