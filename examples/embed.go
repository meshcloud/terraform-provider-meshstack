package examples

import _ "embed"

var (
	//go:embed resources/meshstack_location/resource.tf
	LocationResourceConfig string
	//go:embed data-sources/meshstack_projects/data-source_all.tf
	ProjectsDataSourceConfig string
	//go:embed data-sources/meshstack_projects/data-source_payment_method.tf
	ProjectsWithPaymentMethodDataSourceConfig string
	//go:embed data-sources/meshstack_integrations/data-source.tf
	IntegrationsDataSourceConfig string
	//go:embed data-sources/meshstack_tag_definitions/data-source.tf
	TagDefinitionsDataSourceConfig string
)
