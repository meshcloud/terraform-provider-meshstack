package types

import (
	"github.com/meshcloud/terraform-provider-meshstack/client/types/variant"
)

type (
	SetElem string

	Secret struct {
		Plaintext *string `json:"plaintext,omitempty" tfsdk:"plaintext"`
		Hash      *string `json:"hash,omitempty" tfsdk:"-"`
	}

	SecretOrAny = variant.Variant[Secret, any]

	Any any
)
