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

## Files used by the acceptance tests

Beyond the documented example, a resource/data source directory holds the configs its acceptance
test applies:

* `resource-test-<index>.tf` / `data-source-test-<index>.tf` — the example as one test step applies
  it: same block, but with the documented data sources swapped for resources the test creates and
  names carrying the run's random `var.suffix`. One file per step, numbered from 1; the index is
  what links the file to the step in the test (`examples.Resource.TestStepConfig(t, name, index, …)`).
* `test-support_<name>.tf` — the prerequisites those step files reference (owning workspace, tag
  definitions, …) plus the `variable` declarations the test fills via `ConfigVariables`. A step
  config is its `resource-test-<index>.tf` followed by the support files the test names.

`meshstack_project` is the reference for this layout.

Note that the docs tool globs `resource*.tf` / `data-source*.tf`, so `templates/resources.md.tmpl`
skips any example path containing `-test-`. Without that filter the step configs would show up as
documented examples in the registry. Data sources currently render with the tool's built-in
template, so the first `data-source-test-<index>.tf` also needs a `templates/data-sources.md.tmpl`
carrying the same filter.
