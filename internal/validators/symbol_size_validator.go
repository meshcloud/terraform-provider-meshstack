package validators

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

const symbolMaxDecodedBytes = 100 * 1024

var allowedDataURIPrefixes = []string{
	"data:image/png;base64,",
	"data:image/jpeg;base64,",
	"data:image/jpg;base64,",
	"data:image/gif;base64,",
	"data:image/webp;base64,",
	"data:image/svg+xml;base64,",
}

var _ validator.String = SymbolSize{}

// SymbolSize validates the format and size of a building block definition symbol.
//
// The value must be either:
//   - an http:// or https:// URL, or
//   - a data URI with one of the allowed image prefixes (png, jpeg, jpg, gif, webp, svg+xml).
//
// For data URIs, the base64 payload must be valid and the decoded size must not exceed 100 KiB.
type SymbolSize struct{}

func (v SymbolSize) Description(_ context.Context) string {
	return fmt.Sprintf(
		"Symbol must be an http(s):// URL or a data URI with an allowed image type "+
			"(data:image/png, jpeg, jpg, gif, webp, svg+xml — all ;base64,). "+
			"For data URIs, the decoded image must not exceed %d KiB.",
		symbolMaxDecodedBytes/1024,
	)
}

func (v SymbolSize) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v SymbolSize) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()

	if strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "http://") {
		return
	}

	prefix := ""
	for _, p := range allowedDataURIPrefixes {
		if strings.HasPrefix(value, p) {
			prefix = p
			break
		}
	}

	if prefix == "" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Symbol Format",
			"'spec.symbol' must be a valid image data URI (e.g., data:image/png;base64,..., data:image/svg+xml;base64,...) or an http(s):// URL. "+
				"Non-image data URIs are not permitted.",
		)
		return
	}

	encoded := strings.TrimPrefix(value, prefix)
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(encoded)
	}
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Base64 in Symbol Data URI",
			"'spec.symbol' contains invalid base64 data.",
		)
		return
	}

	if len(decoded) > symbolMaxDecodedBytes {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Symbol Image Too Large",
			fmt.Sprintf(
				"'spec.symbol' image data must not exceed %d KiB after base64 decoding, but was %d KiB (%d bytes).",
				symbolMaxDecodedBytes/1024,
				len(decoded)/1024,
				len(decoded),
			),
		)
	}
}
