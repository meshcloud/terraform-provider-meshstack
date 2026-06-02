resource "meshstack_building_block_definition" "sensitive" {
  metadata = {
    owned_by_workspace = "my-workspace"
  }

  spec = {
    display_name = "BB v2 Sensitive Input Test Definition"
    description  = "Definition with a single STATIC sensitive input for BB v2 read tests"
  }

  version_spec = {
    draft = false

    inputs = {
      static_secret = {
        display_name    = "Static Secret"
        type            = "STRING"
        assignment_type = "STATIC"
        sensitive = {
          argument = {
            secret_value = "super-secret-value"
          }
        }
      }
    }

    implementation = {
      terraform = {
        terraform_version = "1.9.0"
        repository_url    = "https://github.com/example/building-block.git"
      }
    }

    outputs = {}
  }
}
