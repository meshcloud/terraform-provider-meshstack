# The random per-run suffix every test-created name carries, so parallel runs never collide. Exactly
# one file in a composed step config declares it.

variable "suffix" {
  type = string
}
