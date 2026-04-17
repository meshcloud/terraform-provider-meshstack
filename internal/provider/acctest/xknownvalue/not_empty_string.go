package xknownvalue

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
)

// KnownStringWithPrefix returns a Check asserting the value starts with the given prefix.
func KnownStringWithPrefix(prefix string) knownvalue.Check {
	return knownvalue.StringFunc(func(v string) error {
		if !strings.HasPrefix(v, prefix) {
			return fmt.Errorf("expected string with prefix %q, got %q", prefix, v)
		}
		return nil
	})
}

// NotEmptyString returns a Check asserting the value is a non-whitespace string.
// Optional consumer functions can perform additional assertions on the value.
func NotEmptyString(consumers ...func(actualValue string) error) knownvalue.Check {
	return knownvalue.StringFunc(func(v string) error {
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("expected non-empty string after trimming whitespace, but is '%s'", v)
		}
		for _, consumer := range consumers {
			err := consumer(v)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
