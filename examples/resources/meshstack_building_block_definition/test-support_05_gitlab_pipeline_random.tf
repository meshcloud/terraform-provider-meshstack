resource "random_string" "display_name_suffix" {
  length  = 8
  upper   = false
  special = false
  numeric = false
}
