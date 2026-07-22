package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"time"
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

func DoAuthorizedRequest[R any](ctx context.Context, c HttpClient, method string, url *url.URL, options ...RequestOption) (result R, err error) {
	if c.Authorization == nil {
		return result, fmt.Errorf("cannot do authorized request with unconfigured authorization")
	}
	authHeader, err := c.Authorization.Header(ctx, c)
	if err != nil {
		return result, err
	}
	return DoRequest[R](ctx, c, method, url, append(options, withHeader("Authorization", authHeader))...)
}

func DoRequest[R any](ctx context.Context, c HttpClient, method string, url *url.URL, options ...RequestOption) (result R, err error) {
	var body []byte
	body, err = c.doRequest(ctx, method, url, options)
	if err != nil {
		return
	}
	if len(body) == 0 {
		// An empty body is expected only for no-content calls, which are typed DoRequest[any] (e.g.
		// trigger-run, delete) and ignore the result. For a call that expects an object (a pointer or a
		// concrete struct), an empty 2xx body is unexpected — fail loudly instead of returning a nil/zero
		// value that the caller would dereference or mistake for a 404/"not found".
		if t := reflect.TypeFor[R](); t.Kind() == reflect.Interface && t.NumMethod() == 0 {
			return
		}
		err = fmt.Errorf("unexpected empty response body from %s %s", method, url)
		return
	}
	err = json.Unmarshal(body, &result)
	return
}

func (c HttpClient) doRequest(ctx context.Context, method string, url *url.URL, options []RequestOption) ([]byte, error) {
	options = slices.Insert(options, 0,
		withHeader("User-Agent", c.UserAgent),
	)
	opts := requestOptions{}
	for _, option := range options {
		option(&opts)
	}
	if opts.optionErr != nil {
		return nil, opts.optionErr
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
	if len(opts.extraPathElems) > 0 {
		url = *url.JoinPath(opts.extraPathElems...)
	}

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
