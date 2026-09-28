# Whether the building block definition support files leave their version a draft. A flow walking a
# re-draft/re-release dials this; everything else takes the released default.

variable "bbd_draft" {
  type    = bool
  default = false
}
