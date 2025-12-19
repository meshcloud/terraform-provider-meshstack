# Examples

This directory contains examples that are mostly used for documentation, but can also be run/tested manually via the Terraform CLI.

The 
[`terraform-plugin-docs` document generation tool](https://github.com/hashicorp/terraform-plugin-docs) 
looks for files in the following locations by default. 
All other files besides the ones mentioned below are ignored by the documentation tool. 
This is useful for creating examples that can run and/or ar testable even if some parts are not relevant for the documentation.

* `provider/provider.tf` example file for the provider index page
* `data-sources/<full resource name>/data-source.tf` example file for the named data source page
* `resources/<full resource name>/resource.tf` example file for the named data source page
* `resources/<<full resource name>>/import.sh`
