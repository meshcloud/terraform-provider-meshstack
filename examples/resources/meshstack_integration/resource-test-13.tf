# The Azure DevOps integration with its PAT taken from another resource's output, so the value is
# unknown while planning. Guards against "returned a value for the write-only attribute ... during
# planning".

resource "terraform_data" "token" {
  input = "token-from-another-resource"
}

resource "meshstack_integration" "example_azure_devops" {
  metadata = {
    owned_by_workspace = meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name = "Azure DevOps Integration"
    config = {
      azuredevops = {
        base_url     = "https://dev.azure.com"
        organization = "my-organization"

        personal_access_token = provider::meshstack::non_ephemeral_secret(terraform_data.token.output)
      }
    }
  }
}
