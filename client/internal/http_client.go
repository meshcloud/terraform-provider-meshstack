package internal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"slices"
	"time"
)

var (
	errNotFound = errors.New("request failed with status Not Found (404)")
)

// HttpClient wraps [http.Client] with convenient request handling thanks to RequestOption.
type HttpClient struct {
	http.Client
	RootUrl   *url.URL
	UserAgent string

	ApiKey                 string
	ApiSecret              string
	Authorization          string
	AuthorizationExpiresAt time.Time
}

func (c *HttpClient) doRequest(method string, url *url.URL, options ...RequestOption) ([]byte, error) {
	options = slices.Insert(options, 0,
		withHeader("User-Agent", c.UserAgent),
	)
	opts := requestOptions{}
	for _, option := range options {
		option(&opts)
	}
	req, err := c.buildRequest(method, *url, opts)
	if err != nil {
		return nil, err
	}
	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	log.Println(res)
	return c.readBodyAndCheckSuccess(res)
}

func (c *HttpClient) readBodyAndCheckSuccess(res *http.Response) ([]byte, error) {
	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read response body, status code %d: %w", res.StatusCode, err)
	}
	log.Printf("Got response body with %d bytes", len(responseBody))

	if res.StatusCode >= 200 && res.StatusCode <= 299 {
		return responseBody, nil
	}
	var errs []error
	if res.StatusCode == http.StatusNotFound {
		errs = append(errs, errNotFound)
	}
	errs = append(errs,
		fmt.Errorf("request failed with status %d (not 2XX successful)", res.StatusCode),
		fmt.Errorf("error response: %s", string(responseBody)),
	)
	return responseBody, errors.Join(errs...)
}

func (c *HttpClient) buildRequest(method string, url url.URL, opts requestOptions) (*http.Request, error) {
	if len(opts.urlQueryParams) > 0 {
		query := url.Query()
		for k, v := range opts.urlQueryParams {
			query.Set(k, v)
		}
		url.RawQuery = query.Encode()
	}

	var requestBody io.ReadWriter
	if opts.requestPayload != nil {
		requestBody = new(bytes.Buffer)
		if err := json.NewEncoder(requestBody).Encode(opts.requestPayload); err != nil {
			return nil, fmt.Errorf("failed to encode request body payload: %w", err)
		}
	}

	req, err := http.NewRequest(method, url.String(), requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	for _, requestModifier := range opts.requestModifiers {
		requestModifier(req)
	}
	return req, err
}

func unmarshalBody[T any](body []byte, err error) (*T, error) {
	if err != nil {
		return nil, err
	}
	var target T
	if err := json.Unmarshal(body, &target); err != nil {
		return nil, err
	}
	return &target, nil
}
