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

// NewHttpClient creates a new client with an underlying http.Client being a pointer to be modified by WithRetry.
func NewHttpClient(rootUrl *url.URL, userAgent string, auth Authorization) HttpClient {
	return HttpClient{&http.Client{Timeout: 5 * time.Minute}, rootUrl, userAgent, auth}
}

// HttpClient wraps [http.Client] with convenient request handling thanks to RequestOption.
type HttpClient struct {
	*http.Client
	RootUrl       *url.URL
	UserAgent     string
	Authorization Authorization
}

func (c HttpClient) doRequest(ctx context.Context, method string, url *url.URL, options ...RequestOption) ([]byte, error) {
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

func (c HttpClient) doAuthorizedRequest(ctx context.Context, method string, url *url.URL, options ...RequestOption) ([]byte, error) {
	if c.Authorization == nil {
		return nil, fmt.Errorf("authorization is not configured")
	}
	authHeader, err := c.Authorization.Header(ctx, c)
	if err != nil {
		return nil, err
	}
	return c.doRequest(ctx, method, url, append(options, withHeader("Authorization", authHeader))...)
}

func (c HttpClient) readBodyAndCheckSuccess(ctx context.Context, res *http.Response) ([]byte, error) {
	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read response body, status code %d: %w", res.StatusCode, err)
	}
	Log.Debug(ctx, "response", "status", res.StatusCode, "body", loggedBody{bytes.NewBuffer(responseBody)})

	if res.StatusCode >= 200 && res.StatusCode <= 299 {
		return responseBody, nil
	}

	return responseBody, HttpError{
		StatusCode:   res.StatusCode,
		ResponseBody: responseBody,
	}
}

func (c HttpClient) buildRequest(ctx context.Context, method string, url url.URL, opts requestOptions) (*http.Request, error) {
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

// unmarshalBody is a generic helper to unmarshal a JSON response.
// It intentionally takes err as second argument to match doAuthorizedRequest and doRequest signatures.
func unmarshalBody[T any](body []byte, err error) (*T, error) {
	if err != nil {
		return nil, err
	}
	var target T
	if err := json.Unmarshal(body, &target); err != nil {
		return nil, fmt.Errorf("cannot unmarshal body: %w", err)
	}
	return &target, nil
}

type MeshInfo struct {
	Version version.Version `json:"version"`
}

func (c HttpClient) GetMeshInfo(ctx context.Context) (*MeshInfo, error) {
	meshInfoUrl := c.RootUrl.JoinPath("/mesh/info")
	return unmarshalBody[MeshInfo](c.doRequest(ctx, "GET", meshInfoUrl))
}
