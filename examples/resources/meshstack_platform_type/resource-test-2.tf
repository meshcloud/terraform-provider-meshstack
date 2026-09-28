resource "meshstack_platform_type" "example" {
  metadata = {
    name               = "CUSTOM-PT-${upper(var.suffix)}"
    owned_by_workspace = meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name     = "My Custom Platform Updated ${var.suffix}"
    default_endpoint = "https://platform.example.com"
    icon             = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciLz4="
  }
}
