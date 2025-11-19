data "meshstack_payment_method" "example" {
  metadata = {
    name                = "my-payment-method"
    owned_by_workspace  = "my-workspace-identifier"
  }
}
