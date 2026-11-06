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

resource "meshstack_integration" "github" {
  metadata = {
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name = "GitHub Integration"
    config = {
      github = {
        owner           = "my-org"
        base_url        = "https://github.com"
        app_id          = "123456"
        app_private_key = { secret_value = "-----BEGIN RSA PRIVATE KEY-----\nMOCK_KEY_CONTENT\n-----END RSA PRIVATE KEY-----" }
      }
    }
  }
}

resource "meshstack_integration" "github_b" {
  metadata = {
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name = "GitHub Integration B"
    config = {
      github = {
        owner           = "my-org"
        base_url        = "https://github.com"
        app_id          = "123456"
        app_private_key = { secret_value = "-----BEGIN RSA PRIVATE KEY-----\nMOCK_KEY_CONTENT\n-----END RSA PRIVATE KEY-----" }
      }
    }
  }
}

# An example for github_workflows implementation with required attributes only
resource "meshstack_building_block_definition" "example_02_github_workflows" {
  metadata = {
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name = "Example Building Block"
    description  = "An example building block definition"
  }

  version_spec = {
    draft = false

    inputs = {
      workflow_ref = {
        display_name    = "Workflow Reference"
        type            = "STRING"
        assignment_type = "USER_INPUT"
      }
    }

    deletion_mode = "PURGE"
    implementation = {
      github_workflows = {
        repository      = "building-block"
        branch          = "main"
        apply_workflow  = "apply.yml"
        integration_ref = meshstack_integration.github.ref
        # Optional flags, default false
        async                 = true
        omit_run_object_input = true
      }
    }

    outputs = {
      workflow_run_url = {
        display_name    = "Workflow Run URL"
        type            = "STRING"
        assignment_type = "RESOURCE_URL"
      }
    }
  }
  depends_on = [meshstack_integration.github]
}
