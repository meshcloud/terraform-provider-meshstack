data "meshstack_instance" "test" {}

data "meshstack_workspace" "example" {
  metadata = {
    name = data.meshstack_instance.test.admin_workspace_identifier
  }
}

resource "meshstack_integration" "example_entra_id" {
  metadata = {
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name = "Entra ID Integration"
    config = {
      entraid = {
        tenant_id     = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
        client_id     = "yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy"
        client_secret = provider::meshstack::non_ephemeral_secret("updated-plaintext-secret")
      }
    }
  }
}
