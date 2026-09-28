# Whether the building block definition lets a consumer see its run logs. It decides whether a
# workspace-scoped key may trigger a repair run at all.

variable "run_transparency" {
  type    = bool
  default = true
}
