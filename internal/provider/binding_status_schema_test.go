package provider

import (
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/meshcloud/meshstack-cli/client"
)

func expectApprovalAssigned(bindingAddress string) statecheck.StateCheck {
	return statecheck.ExpectKnownValue(bindingAddress, tfjsonpath.New("status").AtMapKey("approval"), knownvalue.StringExact(client.MeshBindingApprovalAssigned.String()))
}

// subjectStateCheck checks an attribute that only bindings of one subject kind have.
type subjectStateCheck func(bindingAddress string) statecheck.StateCheck

func expectUserEnabled(bindingAddress string) statecheck.StateCheck {
	return statecheck.ExpectKnownValue(bindingAddress, tfjsonpath.New("status").AtMapKey("user_disabled"), knownvalue.Bool(false))
}

func subjectStateChecks(bindingAddress string, checks []subjectStateCheck) []statecheck.StateCheck {
	stateChecks := make([]statecheck.StateCheck, len(checks))
	for i, check := range checks {
		stateChecks[i] = check(bindingAddress)
	}
	return stateChecks
}
