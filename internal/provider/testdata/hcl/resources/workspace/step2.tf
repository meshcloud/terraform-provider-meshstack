resource "meshstack_tag_definition" "cost_center" {
  spec = {
    target_kind  = "meshWorkspace"
    key          = "cost-center-[[.suffix]]"
    display_name = "Cost Center"
    value_type = {
      string = {}
    }
  }
}

[[ example "resources/meshstack_workspace/resource.tf" | replace "my-workspace" (print "test-ws-" .suffix) | replace "cost-center" (print "cost-center-" .suffix) ]]
