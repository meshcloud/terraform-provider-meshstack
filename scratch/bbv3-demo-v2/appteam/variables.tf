variable "appteam_client_id" {
  type      = string
  sensitive = true
}

variable "appteam_client_secret" {
  type      = string
  sensitive = true
}

variable "appteam_workspace" {
  type        = string
  description = "App team workspace identifier (bootstrap output); the block's target."
}

variable "suffix" {
  type        = string
  description = "Demo suffix (bootstrap output); used to find the operator's BBD by display name."
}

# Which definition version the app team pins to:
#   v1 -> versions[0] (the broken release it first consumes; step 5)
#   v2 -> versions[1] (after the operator upgraded the block externally; step 9)
variable "pin" {
  type    = string
  default = "v1"
  validation {
    condition     = contains(["v1", "v2"], var.pin)
    error_message = "pin must be one of: v1, v2."
  }
}
