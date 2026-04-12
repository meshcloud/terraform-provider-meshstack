package xknownvalue

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
)

// Ref returns a StateCheck asserting that the resource at resourceAddress has a `ref` attribute
// with the expected kind and a non-empty uuid. If uuidOut is non-nil, captures the uuid for
// subsequent steps to verify it doesn't change.
func Ref(resourceAddress testconfig.Traversal, expectedKind string, uuidOut *string) statecheck.StateCheck {
	return statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("ref"), knownvalue.MapExact(map[string]knownvalue.Check{
		"kind": knownvalue.StringExact(expectedKind),
		"uuid": NotEmptyString(func(actualValue string) error {
			if uuidOut == nil {
				return nil
			}
			if *uuidOut == "" {
				*uuidOut = actualValue
			} else if *uuidOut != actualValue {
				return fmt.Errorf("mismatching Resource UUID %s vs. %s, which should never change", *uuidOut, actualValue)
			}
			return nil
		}),
	}))
}
