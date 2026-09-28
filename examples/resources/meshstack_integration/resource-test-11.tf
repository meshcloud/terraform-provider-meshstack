resource "meshstack_integration" "example_entra_id" {
  metadata = {
    owned_by_workspace = "demo-partner"
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
