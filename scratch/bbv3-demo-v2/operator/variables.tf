variable "operator_client_id" {
  type      = string
  sensitive = true
}

variable "operator_client_secret" {
  type      = string
  sensitive = true
}

variable "platform_workspace" {
  type        = string
  description = "Platform workspace identifier (bootstrap output); owns the BBD and operator key."
}

variable "suffix" {
  type        = string
  description = "Demo suffix (bootstrap output); embedded in the BBD display name."
}

# Drives the BBD lifecycle across the walk-through:
#   draft-good   -> ref_name=main,   draft=true   (working draft; steps 1-2)
#   draft-broken -> ref_name=broken, draft=true   (broken draft; step 3)
#   v1-released  -> ref_name=broken, draft=false  (broken v1 released; step 4)
#   v2-draft     -> ref_name=main,   draft=true   (fixed; creates v2 draft + adds defaulted `size`; step 6a)
#   v2-released  -> ref_name=main,   draft=false  (v2 released; steps 6b, 7, 8)
variable "bbd_phase" {
  type    = string
  default = "draft-good"
  validation {
    condition     = contains(["draft-good", "draft-broken", "v1-released", "v2-draft", "v2-released"], var.bbd_phase)
    error_message = "bbd_phase must be one of: draft-good, draft-broken, v1-released, v2-draft, v2-released."
  }
}

# Enables the cross-workspace adopt+upgrade of the app team's blocks (steps 7-8).
variable "manage_appteam" {
  type    = bool
  default = false
}
