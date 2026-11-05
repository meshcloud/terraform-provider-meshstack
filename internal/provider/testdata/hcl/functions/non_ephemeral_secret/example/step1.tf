resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

locals {
  some_sensitive_value = sensitive("mock-pat-token-12345")
}

resource "meshstack_integration" "example" {
  metadata = {
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }

  spec = {
    display_name = "Azure DevOps Integration"
    config = {
      azuredevops = {
        base_url     = "https://dev.azure.com"
        organization = "my-organization"

        personal_access_token = provider::meshstack::non_ephemeral_secret(nonsensitive(local.some_sensitive_value))
      }
    }
  }
}
