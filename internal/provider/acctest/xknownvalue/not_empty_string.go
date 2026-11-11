package xknownvalue

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
)

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
