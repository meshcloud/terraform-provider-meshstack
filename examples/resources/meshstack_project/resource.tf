resource "meshstack_project" "example" {
  metadata = {
    name               = "my-project-identifier"
    owned_by_workspace = "my-workspace-identifier"
  }
  spec = {
    payment_method_identifier = "my-payment-method-identifier"
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
