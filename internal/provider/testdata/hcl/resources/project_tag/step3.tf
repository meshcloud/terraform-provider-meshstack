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

resource "meshstack_project" "test" {
  metadata = {
    name               = "my-project"
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }
  spec = {
    display_name              = "Test Project"
    payment_method_identifier = meshstack_payment_method.test.metadata.name
  }
}

resource "meshstack_tag_definition" "cost_center" {
  spec = {
    target_kind  = "meshProject"
    key          = "cost-center-[[.suffix]]"
    display_name = "Cost Center"
    value_type = {
      string = {}
    }
  }
}

[[ example "data-sources/meshstack_project/data-source.tf" ]]

data "meshstack_tag_definition" "cost_center" {
  metadata = {
    name = meshstack_tag_definition.cost_center.metadata.name
  }
}

resource "meshstack_project_tag" "example" {
  project_ref = {
    owned_by_workspace = data.meshstack_project.example.metadata.owned_by_workspace
    name               = data.meshstack_project.example.metadata.name
  }
  key    = data.meshstack_tag_definition.cost_center.spec.key
  values = ["12345", "67890"]
}
