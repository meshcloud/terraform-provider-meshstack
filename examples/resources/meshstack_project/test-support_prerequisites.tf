# Prerequisites for every TestAccProject step: the owning workspace, the payment method the project
# bills to and the tag definitions the declared tags reference. They replace the data sources of
# resource.tf, and their names carry the run's random suffix so parallel runs never collide.

variable "suffix" {
  type = string
}

resource "meshstack_workspace" "example" {
  metadata = {
    name = "test-ws-${var.suffix}"
    tags = {
      (meshstack_tag_definition.workspace_tag.spec.key) = ["12345"]
    }
  }
  spec = {
    display_name = "My Workspace's Display Name"
  }
}

resource "meshstack_payment_method" "example" {
  metadata = {
    name               = "test-pm-${var.suffix}"
    owned_by_workspace = meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name    = "My Payment Method"
    expiration_date = "2025-12-31"
    amount          = 10000
    tags = {
      (meshstack_tag_definition.payment_method_tag.spec.key) = ["0000"]
    }
  }
}

resource "meshstack_tag_definition" "workspace_tag" {
  spec = {
    target_kind  = "meshWorkspace"
    key          = "test-key-workspace-${var.suffix}"
    display_name = "Test Tag"

    value_type = {
      string = {}
    }
  }
}

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

resource "meshstack_tag_definition" "project_tag" {
  spec = {
    target_kind  = "meshProject"
    key          = "test-key-project-${var.suffix}"
    display_name = "Test Tag"

    value_type = {
      string = {}
    }
  }
}
