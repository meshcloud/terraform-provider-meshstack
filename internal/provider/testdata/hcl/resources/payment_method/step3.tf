resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

data "meshstack_workspace" "example" {
  metadata = {
    name = meshstack_workspace.test.metadata.name
  }
}

resource "meshstack_tag_definition" "cost_center" {
  spec = {
    target_kind  = "meshPaymentMethod"
    key          = "cost-center-[[.suffix]]"
    display_name = "Cost Center"
    value_type = {
      string = {}
    }
  }
}

resource "meshstack_payment_method" "example" {
  metadata = {
    name               = "test-pm-[[.suffix]]"
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name    = "Updated Payment Method"
    expiration_date = "2025-12-31"
    amount          = 10000
    tags = {
      "cost-center-[[.suffix]]" = ["0000"]
    }
  }
}
