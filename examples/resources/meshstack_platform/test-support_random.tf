resource "random_string" "name_suffix" {
  length  = 8
  upper   = false
  special = false
  numeric = false
}
