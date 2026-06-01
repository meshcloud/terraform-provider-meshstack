package types

import (
	"reflect"
	"strings"

	"github.com/meshcloud/terraform-provider-meshstack/client/types/variant"
)

type (
	Set[T any] []T

	Secret struct {
		// Plaintext is optionally set if secret is initially created (or rotated later)
		Plaintext *string `json:"plaintext,omitempty" tfsdk:"plaintext"`
		// Hash is always present in responses (Plaintext is never returned) and set in requests if secret is supposed to be kept.
		Hash *string `json:"hash,omitempty" tfsdk:"-"`
	}

	SecretOrAny = variant.Variant[Secret, any]

	Any any
)

// IsSet returns true if the given type uses the generic Set type, ignoring the concrete container type T.
func IsSet(other reflect.Type) bool {
	var (
		setType = reflect.TypeFor[Set[any]]()
	)
	if other.PkgPath() == setType.PkgPath() {
		stripGenerics := func(s string) string {
			if startIdx := strings.Index(s, "["); startIdx > 0 {
				return s[0 : startIdx-1]
			}
			return s
		}
		if stripGenerics(other.Name()) == stripGenerics(setType.Name()) {
			return true
		}
	}
	return false
}
