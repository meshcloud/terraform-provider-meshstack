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

resource "meshstack_platform_type" "test" {
  metadata = {
    name               = "TEST-PT-[[.upper_suffix]]"
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }
  spec = {
    display_name     = "Test Platform Type [[.upper_suffix]]"
    default_endpoint = "https://platform.example.com"
    icon             = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciLz4="
  }
}

resource "meshstack_platform" "test" {
  metadata = {
    name               = "test-platform-[[.suffix]]"
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }

  spec = {
    display_name      = "Example Platform"
    description       = "Custom platform using a meshPlatformType"
    endpoint          = "https://custom-platform.example.com"
    documentation_url = "https://docs.example.com"
    location_ref      = { name = "global" }

    availability = {
      restriction              = "RESTRICTED"
      publication_state        = "PUBLISHED"
      restricted_to_workspaces = [meshstack_workspace.test.metadata.name, meshstack_workspace.other.metadata.name]
    }

    quota_definitions = []

    config = {
      custom = {
        platform_type_ref = meshstack_platform_type.test.ref

        metering = {
          processing = {
            enabled = true
          }
        }
      }
    }

    contributing_workspaces = []
  }
}

resource "meshstack_platform" "priv" {
  metadata = {
    name               = "test-priv-platform-[[.suffix]]"
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }

  spec = {
    display_name      = "Example Platform"
    description       = "Custom platform using a meshPlatformType"
    endpoint          = "https://custom-platform.example.com"
    documentation_url = "https://docs.example.com"
    location_ref      = { name = "global" }

    availability = {
      restriction              = "PRIVATE"
      publication_state        = "UNPUBLISHED"
      restricted_to_workspaces = [meshstack_workspace.test.metadata.name]
    }

    quota_definitions = []

    config = {
      custom = {
        platform_type_ref = meshstack_platform_type.test.ref

        metering = {
          processing = {
            enabled = true
          }
        }
      }
    }

    contributing_workspaces = []
  }
}

resource "meshstack_api_key" "test" {
  metadata = {
    owned_by_workspace = meshstack_workspace.other.metadata.name
  }

  spec = {
    display_name = "ci-key"
    permissions  = ["PLATFORMINSTANCE_LIST"]
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

data "meshstack_platforms" "published" {
  publication_state  = "PUBLISHED"
  owned_by_workspace = meshstack_workspace.test.metadata.name
  provider           = meshstack-other
}
