resource "meshstack_integration" "example_azure_devops" {
  metadata = {
    owned_by_workspace = "my-workspace"
  }

  spec = {
    display_name = "Azure DevOps Integration"
    config = {
      azuredevops = {
        base_url     = "https://dev.azure.com"
        organization = "my-organization"
        personal_access_token = {
          secret_value   = "mock-pat-token-12345"
          secret_version = null
        }
      }
    }
  }
}
