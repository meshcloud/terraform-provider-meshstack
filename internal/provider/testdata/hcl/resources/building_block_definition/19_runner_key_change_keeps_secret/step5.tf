resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

resource "meshstack_building_block_runner" "first" {
  metadata = {
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }

  spec = {
    display_name        = "First Runner"
    implementation_type = "TERRAFORM"
    public_key          = <<-EOT
      -----BEGIN PUBLIC KEY-----
      MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAuP1aaqG5oCtnyPUWVbor
      11jsYaOt5A/IzOCA8QsrKFIpMlPOTl9HNsKDDUmYPMNI6uNCpFHmcV46/cqr47s8
      s8NwxckJeHLDItGXGJ8Z2k2YUSp4UPH0PxTNqr5XDIzucxYEoRPluy3jin54B6qU
      gd/BSLJq5e9Ifx00G24FMqJ1Bc24/lYVnBxuXkFHGXJMpfVTgOPlEbIjHYeSfB6U
      QVjUsLddpJCKG/dtQYLrufH3k7A8d6G3PlMe+BNOJpNc9zSxzd6bgbY50Q8BHnmI
      D6/hNv0bNwwerI9eVDrsyDzQ/5YiK+APjAHrzqpCizooYyHgzvqN6aHQEGm4EC0w
      VwIDAQAB
      -----END PUBLIC KEY-----
    EOT
    restriction         = "PRIVATE"
  }
}

resource "meshstack_building_block_runner" "second" {
  metadata = {
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }

  spec = {
    display_name        = "Second Runner"
    implementation_type = "TERRAFORM"
    public_key          = <<-EOT
      -----BEGIN PUBLIC KEY-----
      MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAxBmdjKsT7GJv8DtipzWd
      FvJ/WLJOXpDa5dMf9bAvpxjkn3ahDhD/mZk8oZbEU2fGbVwfEp58/DhK5ZFzfmy7
      Hi6k2D/OBElxvCx5UWqn4W9c47qk59SOHcQmNri7jx/Oh21HZswvlCPpCLr4BmZa
      BzIHmcK+FzeuKIizsM4ULwnSyeIs+meCoLNwwCMSCiWASMDZHbToye2fwGdgCMmE
      xcx+VLuhwRB63ZqTuIM+YgZKhTAoV/QITjjoYlBsIzyC5N79ONOsjcjFqdbQNihd
      qziAZ2W7lFyClEyoyiYb8OFIOTH30JHyGrzMxNERpJjl24+NWAdEp4m80eqBzs5k
      KwIDAQAB
      -----END PUBLIC KEY-----
    EOT
    restriction         = "PRIVATE"
  }
}

resource "meshstack_building_block_runner" "third" {
  metadata = {
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }

  spec = {
    display_name        = "Third Runner"
    implementation_type = "TERRAFORM"
    public_key          = <<-EOT
      -----BEGIN PUBLIC KEY-----
      MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAuP1aaqG5oCtnyPUWVbor
      11jsYaOt5A/IzOCA8QsrKFIpMlPOTl9HNsKDDUmYPMNI6uNCpFHmcV46/cqr47s8
      s8NwxckJeHLDItGXGJ8Z2k2YUSp4UPH0PxTNqr5XDIzucxYEoRPluy3jin54B6qU
      gd/BSLJq5e9Ifx00G24FMqJ1Bc24/lYVnBxuXkFHGXJMpfVTgOPlEbIjHYeSfB6U
      QVjUsLddpJCKG/dtQYLrufH3k7A8d6G3PlMe+BNOJpNc9zSxzd6bgbY50Q8BHnmI
      D6/hNv0bNwwerI9eVDrsyDzQ/5YiK+APjAHrzqpCizooYyHgzvqN6aHQEGm4EC0w
      VwIDAQAB
      -----END PUBLIC KEY-----
    EOT
    restriction         = "PRIVATE"
  }
}

resource "meshstack_building_block_definition" "test" {
  depends_on = [meshstack_building_block_runner.first]

  metadata = {
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }

  spec = {
    display_name = "Runner Key Change [[.suffix]]"
    description  = "Keeps its sensitive input when the runner changes"
  }

  version_spec = {
    draft      = true
    runner_ref = meshstack_building_block_runner.third.ref

    inputs = {
      CONNECTOR_SECRET = {
        display_name    = "Connector Secret"
        type            = "STRING"
        assignment_type = "STATIC"
        sensitive = {
          argument = {
            secret_value   = "plaintext-secret-v1"
            secret_version = "v1"
          }
        }
      }
    }

    implementation = {
      terraform = {
        terraform_version = "1.9.0"
        repository_url    = "https://github.com/example/building-block.git"
        ref_name          = "v2.0.0"
      }
    }
  }
}
