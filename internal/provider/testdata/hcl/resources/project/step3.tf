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

resource "meshstack_payment_method" "test" {
  metadata = {
    name               = "test-pm-[[.suffix]]"
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }
  spec = {
    display_name    = "Test Payment Method"
    expiration_date = "2099-12-31"
  }
}

data "meshstack_payment_method" "example" {
  metadata = {
    name               = meshstack_payment_method.test.metadata.name
    owned_by_workspace = meshstack_payment_method.test.metadata.owned_by_workspace
  }
}

resource "meshstack_tag_definition" "my_key" {
  spec = {
    target_kind  = "meshProject"
    key          = "my-key-[[.suffix]]"
    display_name = "My Key"
    value_type = {
      string = {}
    }
  }
}

data "meshstack_tag_definition" "my_key" {
  metadata = {
    name = meshstack_tag_definition.my_key.metadata.name
  }
}

resource "meshstack_project" "example" {
  metadata = {
    name               = "my-project"
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }
  spec = {
    payment_method_identifier = data.meshstack_payment_method.example.metadata.name
    display_name              = "Updated Display Name"
    tags = {
      (data.meshstack_tag_definition.my_key.spec.key) = [
        "tag-value1",
        "tag-value2",
        "tag-valueN"
      ]
    }
  }
}
