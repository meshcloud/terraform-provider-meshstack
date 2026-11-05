resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

resource "meshstack_tag_definition" "test" {
  spec = {
    target_kind  = "meshPaymentMethod"
    key          = "test-key-[[.suffix]]"
    display_name = "Test Tag"
    value_type = {
      string = {}
    }
  }
}

resource "meshstack_payment_method" "test" {
  metadata = {
    name               = "test-pm-[[.suffix]]"
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }

  spec = {
    display_name    = "My Payment Method"
    expiration_date = "2099-12-31"
    amount          = 10000
    tags = {
      (meshstack_tag_definition.test.spec.key) = ["0000"]
    }
  }
}
