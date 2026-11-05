resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

resource "terraform_data" "token" {
  input = "token-from-another-resource"
}

resource "meshstack_integration" "test" {
  metadata = {
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }

  spec = {
    display_name = "Azure DevOps Integration"
    config = {
      azuredevops = {
        base_url              = "https://dev.azure.com"
        organization          = "my-organization"
        personal_access_token = provider::meshstack::non_ephemeral_secret(terraform_data.token.output)
      }
    }
  }
}
