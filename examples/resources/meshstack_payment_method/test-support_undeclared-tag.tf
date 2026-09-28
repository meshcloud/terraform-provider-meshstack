# A second meshPaymentMethod tag definition the payment method does not declare. The backend still
# returns it as an empty-list entry, so the read path must reconcile it away instead of surfacing it
# as drift.

resource "meshstack_tag_definition" "undeclared_payment_method_tag" {
  spec = {
    target_kind  = "meshPaymentMethod"
    key          = "test-key-undeclared-pm-${var.suffix}"
    display_name = "Test Tag"

    value_type = {
      string = {}
    }
  }
}
