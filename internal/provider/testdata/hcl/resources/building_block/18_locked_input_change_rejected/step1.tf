resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

resource "meshstack_workspace" "other" {
  metadata = {
    name = "test-ws-other-[[.suffix]]"
  }
  spec = {
    display_name = "Other Test Workspace"
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

resource "meshstack_api_key" "consumer" {
  metadata = {
    owned_by_workspace = meshstack_workspace.other.metadata.name
  }

  spec = {
    display_name = "ci-key"
    permissions  = ["BUILDINGBLOCK_SAVE", "BUILDINGBLOCK_LIST", "BUILDINGBLOCK_DELETE"]
  }
}
