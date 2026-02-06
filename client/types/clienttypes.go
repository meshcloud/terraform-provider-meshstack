package types

import (
	"github.com/meshcloud/terraform-provider-meshstack/client/types/variant"
)

type (
	String = string
	Number = int64
	Any    = any

	Secret struct {
		Plaintext *string `json:"plaintext,omitempty" tfsdk:"plaintext"`
		Hash      *string `json:"hash,omitempty" tfsdk:"-"`
	}

	SecretOrAny = variant.Variant[Secret, Any]
)
