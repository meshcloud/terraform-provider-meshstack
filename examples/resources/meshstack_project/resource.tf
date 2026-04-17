resource "meshstack_project" "example" {
  metadata = {
    name               = "my-project"
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }
  spec = {
    payment_method_identifier = data.meshstack_payment_method.example.metadata.name
    display_name              = "My Project's Display Name"
    tags = {
      "tag-key" = [
        "tag-value1",
        "tag-value2",
        "tag-valueN"
      ]
    }
  }
}
