data "meshstack_workspace" "example" {
  metadata = {
    name = "my-workspace-identifier"
  }
}

resource "meshstack_payment_method" "example" {
  metadata = {
    name                = "my-payment-method"
    owned_by_workspace  = data.meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name    = "My Payment Method"
    expiration_date = "2025-12-31T23:59:59Z"
    amount          = 10000
    tags = {
      Country = ["US"]
      Type    = ["production"]
    }
  }
}
