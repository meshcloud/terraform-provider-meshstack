data "meshstack_payment_method" "example" {
  metadata = {
    name               = meshstack_payment_method.example.metadata.name
    owned_by_workspace = meshstack_workspace.example.metadata.name
  }
}
