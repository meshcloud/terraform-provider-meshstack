# The random per-run suffix every test-created name carries, so parallel runs never collide. Exactly
# one file in a composed step config declares it: this one, or a test-support_prerequisites.tf.

variable "suffix" {
  type = string
}
