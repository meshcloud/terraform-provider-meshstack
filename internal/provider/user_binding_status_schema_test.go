package provider

import (
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
)

// subjectStateCheck checks an attribute that only bindings of one subject kind have.
type subjectStateCheck func(bindingAddr testconfig.Traversal) statecheck.StateCheck

func expectUserEnabled(bindingAddr testconfig.Traversal) statecheck.StateCheck {
	return statecheck.ExpectKnownValue(bindingAddr.String(), tfjsonpath.New("status").AtMapKey("user_disabled"), knownvalue.Bool(false))
}

func subjectStateChecks(bindingAddr testconfig.Traversal, checks []subjectStateCheck) []statecheck.StateCheck {
	stateChecks := make([]statecheck.StateCheck, len(checks))
	for i, check := range checks {
		stateChecks[i] = check(bindingAddr)
	}
	return stateChecks
}
