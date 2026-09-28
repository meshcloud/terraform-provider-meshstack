# Prerequisite for the payment method steps: the definition their tag key references. The owning
# workspace comes from the meshstack_workspace example's own step config, which also declares
# `variable "suffix"`.

resource "meshstack_tag_definition" "payment_method_tag" {
  spec = {
    target_kind  = "meshPaymentMethod"
    key          = "test-key-payment-method-${var.suffix}"
    display_name = "Test Tag"

    value_type = {
      string = {}
    }
  }
}
