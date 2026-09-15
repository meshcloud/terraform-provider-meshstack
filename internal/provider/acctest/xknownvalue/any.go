package xknownvalue

import "github.com/hashicorp/terraform-plugin-testing/knownvalue"

// Any accepts every value, null included. It marks a key of a MapExact whose value depends on the
// backend the test runs against, so the map stays exact for the other keys.
func Any() knownvalue.Check {
	return anyValue{}
}

type anyValue struct{}

func (anyValue) CheckValue(any) error { return nil }

func (anyValue) String() string { return "any value" }
