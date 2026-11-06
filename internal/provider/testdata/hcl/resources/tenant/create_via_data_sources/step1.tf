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
    name               = "test-proj-[[.suffix]]"
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }
  spec = {
    display_name              = "Test Project"
    payment_method_identifier = meshstack_payment_method.test.metadata.name
  }
}

resource "meshstack_platform_type" "test" {
  metadata = {
    name               = upper("custom-pt-[[.suffix]]")
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }
  spec = {
    display_name     = "Test Platform Type [[.suffix]]"
    default_endpoint = "https://platform.example.com"
    icon             = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciLz4="
  }
}

resource "meshstack_platform" "test" {
  metadata = {
    name               = "custom-[[.suffix]]"
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }
  spec = {
    display_name      = "Test Platform"
    description       = "Custom platform using a meshPlatformType"
    endpoint          = "https://custom-platform.example.com"
    documentation_url = "https://docs.example.com"
    location_ref      = { name = "global" }
    availability = {
      restriction              = "PUBLIC"
      publication_state        = "PUBLISHED"
      restricted_to_workspaces = []
    }
    quota_definitions = []
    config = {
      custom = {
        platform_type_ref = meshstack_platform_type.test.ref
      }
    }
    contributing_workspaces = []
  }
}

resource "meshstack_building_block_definition" "platform_tenant_id" {
  metadata = {
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }
  spec = {
    display_name        = "Platform Tenant ID Provider"
    description         = "Provides a platform tenant ID for custom platform tenants"
    target_type         = "TENANT_LEVEL"
    supported_platforms = [{ name = meshstack_platform_type.test.metadata.name }]
  }
  version_spec = {
    draft = false
    inputs = {
      tenant_id = {
        display_name    = "Tenant ID"
        type            = "STRING"
        assignment_type = "STATIC"
        argument        = jsonencode("test-tenant-id")
      }
    }
    implementation = {
      manual = {}
    }
    outputs = {
      tenant_id = {
        display_name    = "Tenant ID"
        assignment_type = "PLATFORM_TENANT_ID"
      }
    }
  }
}

resource "meshstack_landingzone" "test" {
  metadata = {
    name               = "test-lz-[[.suffix]]"
    owned_by_workspace = meshstack_workspace.test.metadata.name
    tags               = {}
  }
  spec = {
    display_name                  = "Test Landing Zone"
    description                   = "A custom landing zone"
    automate_deletion_approval    = false
    automate_deletion_replication = false
    info_link                     = "https://example.com"
    platform_ref                  = meshstack_platform.test.ref
    platform_properties = {
      custom = {}
    }
    mandatory_building_block_refs = [meshstack_building_block_definition.platform_tenant_id.ref]
  }
}

data "meshstack_platform" "test" {
  metadata = {
    uuid = meshstack_platform.test.metadata.uuid
  }
}

data "meshstack_platforms" "test" {
  publication_state  = "PUBLISHED"
  owned_by_workspace = meshstack_platform.test.metadata.owned_by_workspace
}

data "meshstack_landingzone" "test" {
  metadata = {
    name = meshstack_landingzone.test.metadata.name
  }
}

data "meshstack_landingzones" "test" {
  platform_uuid = meshstack_landingzone.test.spec.platform_ref.uuid
  restricted    = false
}

resource "meshstack_tenant" "test" {
  metadata = {
    owned_by_workspace = meshstack_project.test.metadata.owned_by_workspace
    owned_by_project   = meshstack_project.test.metadata.name
  }
  spec = {
    platform_ref     = one([for p in data.meshstack_platforms.test.platforms : p if p.metadata.uuid == meshstack_platform.test.metadata.uuid]).ref
    landing_zone_ref = data.meshstack_landingzone.test.ref
  }
  wait_for_completion = true
}
