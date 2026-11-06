data "meshstack_instance" "test" {}

data "meshstack_workspace" "example" {
  metadata = {
    name = data.meshstack_instance.test.admin_workspace_identifier
  }
}

[[ example "resources/meshstack_integration/resource_04_entra_id.tf" ]]
