resource "meshstack_project" "example" {
  metadata = {
    name               = "my-project"
    owned_by_workspace = "my-workspace"
  }
  spec = {
    payment_method_identifier = "my-payment-method"
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
