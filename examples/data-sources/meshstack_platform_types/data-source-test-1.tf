data "meshstack_platform_types" "all" {
  # depends_on, because the listing references no attribute of the platform type: without it the
  # query could run first and read an empty store.
  depends_on = [meshstack_platform_type.example]
}
