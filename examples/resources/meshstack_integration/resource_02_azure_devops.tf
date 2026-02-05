resource "meshstack_integration" "example_azure_devops" {
  metadata = {
    owned_by_workspace = "my-workspace"
  }

  spec = {
    display_name = "Azure DevOps Integration"
    config = {
      azuredevops = {
        base_url              = "https://dev.azure.com"
        organization          = "my-organization"
        personal_access_token = "mock-pat-token-12345"
        runner_ref = {
          uuid = "05cfa85f-2818-4bdd-b193-620e0187d7de"
        }
      }
    }
  }
}
