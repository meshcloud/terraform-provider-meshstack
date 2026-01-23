package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/meshcloud/terraform-provider-meshstack/client/version"
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
