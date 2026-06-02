resource "meshstack_building_block_definition" "sensitive_user_input" {
  metadata = {
    owned_by_workspace = "my-workspace"
  }

  spec = {
    display_name = "BB v2 Sensitive User Input Test Definition"
    description  = "Definition with sensitive USER_INPUTs for BB v2 write tests"
  }

  version_spec = {
    draft = false

    inputs = {
      secret_str = {
        display_name    = "Secret String"
        type            = "STRING"
        assignment_type = "USER_INPUT"
        sensitive       = {}
      }
      secret_code = {
        display_name    = "Secret Code"
        type            = "CODE"
        assignment_type = "USER_INPUT"
        sensitive       = {}
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
