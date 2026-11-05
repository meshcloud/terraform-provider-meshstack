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

resource "meshstack_platform" "test" {
  metadata = {
    name               = "k8s-[[.suffix]]"
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }
  spec = {
    display_name      = "Test Platform"
    description       = "Kubernetes Cluster"
    endpoint          = "https://k8s.dev.eu-de-central.msh.host:6443"
    documentation_url = "https://kubernetes.io"
    location_ref      = { name = "global" }
    availability = {
      restriction              = "PUBLIC"
      publication_state        = "PUBLISHED"
      restricted_to_workspaces = []
    }
    quota_definitions = [
      {
        quota_key               = "limits.cpu"
        min_value               = 1
        max_value               = 8000
        unit                    = "cores"
        auto_approval_threshold = 2000
        description             = "vCPU limit"
        label                   = "CPU"
      },
    ]
    config = {
      kubernetes = {
        base_url = "https://k8s.example.com:6443"
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
    platform_ref                  = { uuid = meshstack_platform.test.metadata.uuid }
    platform_properties = {
      kubernetes = {
        kubernetes_role_mappings = []
      }
    }
  }
}

resource "meshstack_tenant" "test" {
  metadata = {
    owned_by_workspace = meshstack_project.test.metadata.owned_by_workspace
    owned_by_project   = meshstack_project.test.metadata.name
  }
  spec = {
    platform_ref     = meshstack_platform.test.ref
    landing_zone_ref = meshstack_landingzone.test.ref
    requested_quotas = {
      "limits.cpu" = { value = 4000 }
    }
  }
  wait_for_completion = true
}
