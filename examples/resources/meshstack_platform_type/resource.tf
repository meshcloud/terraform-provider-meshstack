resource "meshstack_platform_type" "example" {
  metadata = {
    name               = "MY-PLATFORM-TYPE"
    owned_by_workspace = "my-workspace-identifier"
  }

  spec = {
    display_name     = "My Custom Platform"
    default_endpoint = "https://platform.example.com"
    icon             = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciLz4="
  }
}
