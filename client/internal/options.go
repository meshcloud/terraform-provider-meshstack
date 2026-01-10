package internal

import (
	"fmt"
	"net/http"
)

type (
	// RequestOption is a functional option for configuring HTTP requests.
	RequestOption func(opts *requestOptions)

	requestOptions struct {
		urlQueryParams   map[string]string
		requestPayload   any
		requestModifiers []requestModifier
	}
	requestModifier func(req *http.Request)
)

// WithUrlQuery adds a URL query parameter to the request.
// The value is stringified using fmt.Stringer.String() if implemented, otherwise fmt.Sprintf("%v", value).
func WithUrlQuery(key string, value any) RequestOption {
	return func(opts *requestOptions) {
		var valueStr string
		if stringerValue, ok := value.(fmt.Stringer); ok {
			valueStr = stringerValue.String()
		} else {
			valueStr = fmt.Sprintf("%v", value)
		}
		if opts.urlQueryParams == nil {
			opts.urlQueryParams = map[string]string{}
		}
		opts.urlQueryParams[key] = valueStr
	}
}

func appendRequestModifier(modifier requestModifier) RequestOption {
	return func(opts *requestOptions) {
		opts.requestModifiers = append(opts.requestModifiers, modifier)
	}
}

func withAccept(accept string) RequestOption {
	return withHeader("Accept", accept)
}

func withHeader(key, value string) RequestOption {
	return appendRequestModifier(func(req *http.Request) {
		req.Header.Set(key, value)
	})
}

func withPayload(payload any, contentType string) RequestOption {
	return func(opts *requestOptions) {
		withAccept(contentType)(opts)
		withHeader("Content-Type", contentType)(opts)
		opts.requestPayload = payload
	}
}
