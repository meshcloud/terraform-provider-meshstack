resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

data "meshstack_workspace" "example" {
  metadata = {
    name = meshstack_workspace.test.metadata.name
  }
}

resource "meshstack_integration" "example_azure_devops" {
  metadata = {
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name = "Azure DevOps Integration"
    config = {
      azuredevops = {
        base_url              = "https://dev.azure.com"
        organization          = "my-organization"
        personal_access_token = provider::meshstack::non_ephemeral_secret("updated-plaintext-secret")
      }
    }
  }
}
