package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
)

type (
	// RequestOption is a functional option for configuring HTTP requests.
	RequestOption func(opts *requestOptions)

	requestOptions struct {
		urlQueryParams   map[string]string
		extraPathElems   []string
		requestPayload   any
		requestModifiers []requestModifier
		// optionErr holds the first error produced while applying options (e.g. an unmarshalable
		// query); doRequest surfaces it instead of building a request from partial options.
		optionErr error
	}
	requestModifier func(req *http.Request)
)

// WithUrlQuery adds URL query parameters from a query value.
//
// The value is JSON-marshalled and decoded into a flat map, so each field becomes a query param
// named by its `json` tag. A struct passed by value is the common case: its zero-value fields are
// dropped (an implicit `omitempty`), so an unset filter needs neither a pointer nor an `omitempty`
// tag and a zero-value struct adds no params at all. A map[string]string / map[string]any is taken
// verbatim — every entry is sent, including deliberate zero values such as page=0.
//
// Values are stringified with fmt.Sprintf("%v", ...); nested objects or arrays are not supported.
func WithUrlQuery(query any) RequestOption {
	return func(opts *requestOptions) {
		data, err := json.Marshal(query)
		if err != nil {
			opts.optionErr = fmt.Errorf("cannot marshal url query of type %T: %w", query, err)
			return
		}
		// UseNumber keeps integers (e.g. page) from becoming float64 and gaining a ".0" or exponent.
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		var params map[string]any
		if err := decoder.Decode(&params); err != nil {
			opts.optionErr = fmt.Errorf("cannot decode url query of type %T into a flat map: %w", query, err)
			return
		}
		// Drop zero-value fields only for a struct (passed by value, not by pointer); a map is
		// passed through as given.
		skipZero := reflect.ValueOf(query).Kind() == reflect.Struct
		for key, value := range params {
			if value == nil || (skipZero && reflect.ValueOf(value).IsZero()) {
				continue
			}
			if opts.urlQueryParams == nil {
				opts.urlQueryParams = map[string]string{}
			}
			opts.urlQueryParams[key] = fmt.Sprintf("%v", value)
		}
	}
}

// WithPathElems appends path elements to the request URL path.
func WithPathElems(pathElems ...string) RequestOption {
	return func(opts *requestOptions) {
		opts.extraPathElems = append(opts.extraPathElems, pathElems...)
	}
}

func appendRequestModifier(modifier requestModifier) RequestOption {
	return func(opts *requestOptions) {
		opts.requestModifiers = append(opts.requestModifiers, modifier)
	}
}

func WithAccept(accept string) RequestOption {
	return withHeader("Accept", accept)
}

func withHeader(key, value string) RequestOption {
	return appendRequestModifier(func(req *http.Request) {
		req.Header.Set(key, value)
	})
}

func withPayload(payload any, contentType string) RequestOption {
	return func(opts *requestOptions) {
		WithAccept(contentType)(opts)
		withHeader("Content-Type", contentType)(opts)
		opts.requestPayload = payload
	}
}
