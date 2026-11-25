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
    display_name       = "Test Project"
    payment_method_ref = meshstack_payment_method.test.ref
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
  }
}

resource "meshstack_tenant" "example" {
  metadata = {
    owned_by_workspace = meshstack_workspace.test.metadata.name
    owned_by_project   = meshstack_project.test.metadata.name
  }

  spec = {
    platform_ref     = meshstack_platform.test.ref
    landing_zone_ref = meshstack_landingzone.test.ref
  }

  wait_for_completion = true

  timeouts = {
    create = "1s"
  }

  lifecycle {
    postcondition {
      condition     = self.spec.platform_tenant_id != null
      error_message = "Tenant ${self.metadata.uuid} has no platform tenant ID yet."
    }
  }
}
