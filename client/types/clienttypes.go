package types

type (
	String = string
	Number = int64
	Any    = any

	Secret struct {
		Plaintext *string `json:"plaintext,omitempty" tfsdk:"plaintext"`
		Hash      *string `json:"hash,omitempty" tfsdk:"-"`
	}
)
