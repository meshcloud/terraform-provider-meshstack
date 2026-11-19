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

variable "apikey_client_id" {
  type     = string
  nullable = false
}

variable "apikey_client_secret" {
  type      = string
  nullable  = false
  sensitive = true
}

provider "meshstack-other" {
  apikey    = var.apikey_client_id
  apisecret = var.apikey_client_secret
}

resource "meshstack_building_block" "example_workspace" {
  spec = {
    building_block_definition_version_ref = { uuid = meshstack_building_block_definition.example.version_latest_release.uuid }

    display_name = "my-workspace-building-block"
    target_ref   = meshstack_workspace.other.ref

    inputs = {
      environment = {
        value = jsonencode("[[.environment]]")
      }
    }
  }

  provider   = meshstack-other
  depends_on = [meshstack_api_key.consumer]
}
