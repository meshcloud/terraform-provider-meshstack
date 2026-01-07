resource "random_string" "tag_suffix" {
  length  = 8
  upper   = false
  special = false
  numeric = false
}
