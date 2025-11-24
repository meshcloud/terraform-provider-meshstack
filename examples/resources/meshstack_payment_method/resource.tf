data "meshstack_workspace" "example" {
  metadata = {
    name = "my-workspace-identifier"
  }
}

resource "meshstack_payment_method" "example" {
  metadata = {
    name               = "my-payment-method"
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name    = "My Payment Method"
    expiration_date = "2025-12-31"
    amount          = 10000
    tags = {
      CostCenter = ["0000"]
      Type    = ["production"]
    }
  }
}
