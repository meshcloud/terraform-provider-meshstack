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

resource "meshstack_tag_definition" "environment" {
  spec = {
    target_kind  = "meshBuildingBlockDefinition"
    key          = "environment-[[.suffix]]"
    display_name = "Environment"

    value_type = {
      multi_select = {
        options = ["dev", "prod"]
      }
    }
  }
}

resource "meshstack_tag_definition" "cost_center" {
  spec = {
    target_kind  = "meshBuildingBlockDefinition"
    key          = "cost-center-[[.suffix]]"
    display_name = "Cost Center"

    value_type = {
      string = {}
    }
  }
}

# The workspace tag that the building block definition reads as a TAG input.
resource "meshstack_tag_definition" "workspace_business_unit" {
  spec = {
    target_kind  = "meshWorkspace"
    key          = "business-unit-[[.suffix]]"
    display_name = "Business Unit"

    value_type = {
      string = {}
    }
  }
}

# A minimal manual BBD used as a dependency for the terraform example
resource "meshstack_building_block_definition" "other" {
  metadata = {
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name = "Dependency Building Block"
    description  = "A building block that is used as a dependency"
  }

  version_spec = {
    draft = false
    implementation = {
      manual = {}
    }
  }
}

data "meshstack_building_block_definition" "example" {
  metadata = {
    uuid = meshstack_building_block_definition.other.metadata.uuid
  }
}
