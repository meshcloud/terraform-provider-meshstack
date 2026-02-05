resource "meshstack_integration" "example_gitlab" {
  metadata = {
    owned_by_workspace = "my-workspace"
  }

  spec = {
    display_name = "GitLab Integration"
    config = {
      gitlab = {
        base_url = "https://gitlab.com"
        runner_ref = {
          uuid = "f4f4402b-f54d-4ab9-93ae-c07e997041e9"
        }
      }
    }
  }
}
