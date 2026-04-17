package xknownvalue

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
)

// MapExact returns a Check for asserting equality between the
// supplied map[string]Check and the value passed to the CheckValue method.
// It constructs an AI-friendly error message such that AI is able to fix the test assertion automatically.
func MapExact(value map[string]knownvalue.Check) knownvalue.Check {
	return mapExact{
		value: value,
	}
}

type mapExact struct {
	value map[string]knownvalue.Check
}

func (v mapExact) CheckValue(other any) error {
	otherVal, ok := other.(map[string]any)
	if !ok {
		return fmt.Errorf("expected %T value for MapExact check, got: %T", otherVal, other)
	}

	expectedKeys := slices.Collect(maps.Keys(v.value))
	actualKeys := slices.Collect(maps.Keys(otherVal))
	slices.Sort(expectedKeys)
	slices.Sort(actualKeys)

	if slices.Compare(actualKeys, expectedKeys) != 0 {
		missingExpectedKeys := slices.DeleteFunc(slices.Clone(expectedKeys), func(e string) bool {
			return slices.Contains(actualKeys, e)
		})
		missingActualKeys := slices.DeleteFunc(slices.Clone(actualKeys), func(e string) bool {
			return slices.Contains(expectedKeys, e)
		})
		return fmt.Errorf("map elements do not match: missing expected keys %s, missing actual keys %s", missingExpectedKeys, missingActualKeys)
	}

	for _, k := range actualKeys {
		if err := v.value[k].CheckValue(otherVal[k]); err != nil {
			return fmt.Errorf("%s map element: %s", k, err)
		}
	}

	return nil
}

func (v mapExact) String() string {
	var result []string
	for _, key := range slices.Sorted(maps.Keys(v.value)) {
		result = append(result, fmt.Sprintf("%s=%s", key, v.value[key]))
	}
	return strings.Join(result, ", ")
}
