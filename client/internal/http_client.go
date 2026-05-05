package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/meshcloud/terraform-provider-meshstack/client/version"
)

// HttpError represents an HTTP error response with status code.
// This error is returned when an HTTP request fails with a non-2XX status code.
type HttpError struct {
	StatusCode int
	Message    string
}

func (e HttpError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

// IsForbidden returns true if the error is a 403 Forbidden response.
func (e HttpError) IsForbidden() bool {
	return e.StatusCode == http.StatusForbidden
}

// IsNotFound returns true if the error is a 404 Not Found response.
func (e HttpError) IsNotFound() bool {
	return e.StatusCode == http.StatusNotFound
}

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

func (c *HttpClient) doRequest(ctx context.Context, method string, url *url.URL, options ...RequestOption) ([]byte, error) {
	options = slices.Insert(options, 0,
		withHeader("User-Agent", c.UserAgent),
	)
	opts := requestOptions{}
	for _, option := range options {
		option(&opts)
	}
	req, err := c.buildRequest(ctx, method, *url, opts)
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
	return c.readBodyAndCheckSuccess(ctx, res)
}

func (c *HttpClient) readBodyAndCheckSuccess(ctx context.Context, res *http.Response) ([]byte, error) {
	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read response body, status code %d: %w", res.StatusCode, err)
	}
	Log.Debug(ctx, "response", "status", res.StatusCode, "body", loggedBody{bytes.NewBuffer(responseBody)})

	if res.StatusCode >= 200 && res.StatusCode <= 299 {
		return responseBody, nil
	}

	return responseBody, HttpError{
		StatusCode: res.StatusCode,
		Message:    string(responseBody),
	}
}

func (c *HttpClient) buildRequest(ctx context.Context, method string, url url.URL, opts requestOptions) (*http.Request, error) {
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

	req, err := http.NewRequestWithContext(ctx, method, url.String(), requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	for _, requestModifier := range opts.requestModifiers {
		requestModifier(req)
	}
	Log.Debug(ctx, "request", "url", req.URL.String(), "method", req.Method, "headers", loggedHeaders(req.Header), "body", loggedBody{requestBody})
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

type MeshInfo struct {
	Version version.Version `json:"version"`
}

func (c *HttpClient) GetMeshInfo(ctx context.Context) (*MeshInfo, error) {
	meshInfoUrl := c.RootUrl.JoinPath("/mesh/info")
	return unmarshalBody[MeshInfo](c.doRequest(ctx, "GET", meshInfoUrl))
}
