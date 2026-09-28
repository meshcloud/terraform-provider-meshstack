data "meshstack_tenants" "example" {
  workspace = meshstack_tenant.example.metadata.owned_by_workspace
  project   = meshstack_tenant.example.metadata.owned_by_project
}
