resource "meshstack_payment_method" "example" {
  metadata = {
    name               = "my-payment-method"
    owned_by_workspace = "my-workspace"
  }

  spec = {
    display_name    = "My Payment Method"
    expiration_date = "2025-12-31"
    amount          = 10000
    tags = {
      "cost-center" = ["0000"]
    }
  }
}
