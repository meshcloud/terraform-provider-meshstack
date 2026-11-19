resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

resource "meshstack_building_block_definition" "example" {
  metadata = {
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }

  spec = {
    display_name = "Test BB v3 Definition"
    description  = "A building block definition for BB v3 resource tests"
  }

  version_spec = {
    draft = false

    inputs = {
      name = {
        display_name           = "Name"
        type                   = "STRING"
        assignment_type        = "USER_INPUT"
        updateable_by_consumer = true
      }
      environment = {
        display_name           = "Environment"
        type                   = "STRING"
        assignment_type        = "USER_INPUT"
        updateable_by_consumer = false
      }
    }

    implementation = {
      manual = {}
    }

    deletion_mode = "DELETE"
  }
}

resource "meshstack_building_block" "example_workspace" {
  spec = {
    building_block_definition_version_ref = meshstack_building_block_definition.example.version_latest

    display_name = "my-workspace-building-block"
    target_ref   = meshstack_workspace.test.ref

    inputs = {
      name = {
        value = jsonencode("[[.name]]")
      }
      environment = {
        value = jsonencode("[[.environment]]")
      }
    }
  }
}
