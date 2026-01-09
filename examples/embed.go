package examples

import _ "embed"

var (
	//go:embed resources/meshstack_location/resource.tf
	LocationResourceConfig string
	//go:embed data-sources/meshstack_projects/data-source_all.tf
	ProjectsDataSourceConfig string
	//go:embed data-sources/meshstack_projects/data-source_payment_method.tf
	ProjectsWithPaymentMethodDataSourceConfig string
)
