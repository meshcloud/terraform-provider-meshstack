# What resource.tf and data-source.tf look up, under the names their documentation uses, so
# TestAccProjectExample applies both exactly as published. Those names cannot carry a random suffix:
# two runs against one meshStack at once collide, and a failed cleanup needs the local backend rebuilt.

resource "meshstack_workspace" "documented_example" {
  # resource.tf names its tag key literally, so only this orders the tag definition before the
  # project, and its deletion after it.
  depends_on = [meshstack_tag_definition.documented_example]

  metadata = {
    name = "my-workspace"
  }
  spec = {
    display_name = "My Workspace's Display Name"
  }
}

data "meshstack_workspace" "example" {
  metadata = {
    name = meshstack_workspace.documented_example.metadata.name
  }
}

resource "meshstack_payment_method" "documented_example" {
  metadata = {
    name               = "my-payment-method"
    owned_by_workspace = meshstack_workspace.documented_example.metadata.name
  }

  spec = {
    display_name    = "My Payment Method"
    expiration_date = "2025-12-31"
    amount          = 10000
  }
}

data "meshstack_payment_method" "example" {
  metadata = {
    name               = meshstack_payment_method.documented_example.metadata.name
    owned_by_workspace = meshstack_payment_method.documented_example.metadata.owned_by_workspace
  }
}

resource "meshstack_tag_definition" "documented_example" {
  spec = {
    target_kind  = "meshProject"
    key          = "tag-key"
    display_name = "Tag Key"

    value_type = {
      string = {}
    }
  }
}
