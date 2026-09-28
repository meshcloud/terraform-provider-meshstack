# The quota bounds a tenant case exercises. They are variables rather than separate step files
# because only the numbers differ between the quota cases — the blocks are identical.
#
# lz_memory_default at 0 means "no limits.memory quota at all"; a non-zero value defines one on the
# platform and makes it a landing zone default, which the tenant never requests, so the backend
# merges it in and status.applied_quotas becomes a strict superset of spec.requested_quotas.

variable "max_cpu" {
  type    = number
  default = 4000
}

variable "cpu_auto_approval_threshold" {
  type    = number
  default = 4000
}

variable "requested_cpu" {
  type    = number
  default = 2000
}

variable "lz_memory_default" {
  type    = number
  default = 0
}
