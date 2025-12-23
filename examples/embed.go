package examples

import _ "embed"

var (
	//go:embed resources/meshstack_location/resource.tf
	LocationResourceConfig string
)
