resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-{{.suffix}}"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

resource "meshstack_project" "example" {
  metadata = {
    name               = "test-proj-{{.suffix}}"
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }
  spec = {
    display_name = "Test Project"
  }
}

resource "meshstack_workspace_{{.subject_kind}}_binding" "example" {
  metadata = {
    name = "test-wb-{{.suffix}}"
  }
  role_ref = {
    name = "Workspace Member"
  }
  target_ref = {
    name = meshstack_workspace.test.metadata.name
  }
  subject = {
    name = "{{.subject_name}}"
  }
}

{{ example "resources/meshstack_project_" .subject_kind "_binding/resource.tf" }}
