# A browser login acts in one workspace per provider block, so manage several workspaces with an
# aliased block each, as with one `azurerm` block per subscription. Your login needs a role in each.
provider "meshstack" {
  alias     = "team_a"
  workspace = "team-a-ab12c"
}

provider "meshstack" {
  alias     = "team_b"
  workspace = "team-b-cd34e"
}

resource "meshstack_building_block" "team_a_sandbox" {
  provider = meshstack.team_a
  # ...
}
