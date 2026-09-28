# The scalars the building block step files dial between the steps of one flow. Each one defaults to
# the value the documented example carries, so a config that only wants the example's behaviour
# composes this file without passing anything.

variable "bb_display_name" {
  type    = string
  default = "my-workspace-building-block"
}

variable "bb_name" {
  type    = string
  default = "my-name"
}

variable "bb_environment" {
  type    = string
  default = "dev"
}

# Null means the version ref carries no content_hash at all, which is what every step that is not
# walking a definition content change wants.
variable "bb_content_hash" {
  type    = string
  default = null
}
