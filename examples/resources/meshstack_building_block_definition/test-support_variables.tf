# The attributes a definition case dials rather than restructures. They are variables, not separate
# step files, because a released-vs-draft version or a rotated secret is one scalar changing while
# the rest of the definition stays byte-identical — a step file per value would differ by a line.
#
# A structural difference (a different implementation type, an integration swapped, an output
# declared) is a step file of its own instead, so the applied config still reads as plain HCL.

variable "description" {
  type    = string
  default = "An example building block definition"
}

variable "draft" {
  type    = bool
  default = true
}

# The per-run suffix the tag keys carry, so parallel runs never collide on a tag definition.
variable "tag_suffix" {
  type    = string
  default = ""
}

# The sensitive STATIC input's secret. Rotating it means giving these two a new pair of values:
# meshStack re-applies the secret when the version changes, which is what the rotation cases turn on.
variable "secret_value" {
  type      = string
  sensitive = true
  default   = "plaintext-secret-v1"
}

variable "secret_version" {
  type    = string
  default = "v1"
}

# Naming each ordered building block after its inputs. An empty string and a missing attribute mean
# the same thing to meshStack, but Terraform tells them apart, so a case walks both.
variable "display_name_template" {
  type    = string
  default = null
}

# The terraform implementation's pre-run script, changed to prove a re-draft leaves the released
# version's content hash untouched.
variable "pre_run_script" {
  type    = string
  default = null
}

# The approval gates and the drift schedule. Which of these meshStack accepts depends on the
# implementation the stored version carries, so a case dials them alongside an implementation swap.
variable "approval_policies" {
  type    = any
  default = {}
}

variable "schedule" {
  type    = any
  default = null
}
