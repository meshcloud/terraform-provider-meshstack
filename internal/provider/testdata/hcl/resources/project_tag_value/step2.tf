resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
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

resource "meshstack_tag_definition" "teams" {
  spec = {
    target_kind  = "meshProject"
    key          = "teams-[[.suffix]]"
    display_name = "Teams"
    value_type = {
      multi_select = {
        options = ["team-a", "team-b", "team-c"]
      }
    }
  }
}

data "meshstack_tag_definition" "teams" {
  metadata = {
    name = meshstack_tag_definition.teams.metadata.name
  }
}

data "meshstack_workspace" "example" {
  metadata = {
    name = meshstack_workspace.test.metadata.name
  }
}

[[ example "data-sources/meshstack_project/data-source.tf" ]]

[[ example "resources/meshstack_project_tag_value/resource.tf" ]]
