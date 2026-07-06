package internal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// WithRetry sets up the given client to retry certain requests.
// The idempotent methods GET, PUT and DELETE are retried by default, POST only if the path is
// explicitly whitelisted. See RetryOptions.
func WithRetry(c HttpClient, options RetryOptions) HttpClient {
	next := http.DefaultTransport
	if c.Transport != nil {
		next = c.Transport
	}
	whitelistedByMethodAndUrl := func() (m map[string]*sync.Map) {
		m = make(map[string]*sync.Map)
		for method, paths := range options.WhitelistedPaths {
			m[method] = new(sync.Map)
			for _, path := range paths {
				m[method].Store(c.RootUrl.JoinPath(path).String(), nil)
			}
		}
		return
	}()
	c.Transport = &retryRoundTripper{
		Next:       next,
		MaxRetries: options.MaxRetries,
		// ShouldRetryRequest checks if the request method/path is eligible for retry.
		ShouldRetryRequest: func(req *http.Request) (retry bool) {
			if options.Backoff == nil {
				return false
			}
			switch req.Method {
			case http.MethodGet, http.MethodPut, http.MethodDelete:
				// Idempotent methods are safe to retry: replaying them cannot create duplicate
				// side effects. A DELETE that actually succeeded server-side before a proxy 503
				// simply yields a 404 on replay, which delete handlers already treat as done.
				return true
			}
			if whitelisted, found := whitelistedByMethodAndUrl[req.Method]; found {
				_, retry = whitelisted.Load(req.URL.String())
			}
			return
		},
		// ShouldRetryResponse returns the backoff policy if the response/error indicates a retryable condition,
		// otherwise nil is returned to indicate no retry.
		ShouldRetryResponse: func(resp *http.Response, err error) RetryBackoff {
			if err != nil {
				return options.Backoff
			}
			switch resp.StatusCode {
			case http.StatusTooManyRequests, http.StatusServiceUnavailable:
				return retryAfterBackoff{Response: resp, Fallback: options.Backoff}
			case http.StatusBadGateway, http.StatusGatewayTimeout:
				return options.Backoff
			case http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
				if locationRedirectUrl, _ := resp.Request.URL.Parse(resp.Header.Get("Location")); locationRedirectUrl != nil {
					if whitelisted, found := whitelistedByMethodAndUrl[resp.Request.Method]; found {
						whitelisted.Store(locationRedirectUrl.String(), nil)
					}
				}
				return nil
			default:
				return nil
			}
		},
	}
	return c // for fluent API
}

// RetryOptions configure WithRetry.
type RetryOptions struct {
	// MaxRetries limits the attempts to retries. If zero, retries will never be attempted.
	MaxRetries int
	// Backoff to use when retrying. If nil, retries will never be attempted.
	Backoff RetryBackoff
	// WhitelistedPaths allow methods beyond GET and PUT to be retried as well, see WithRetry.
	WhitelistedPaths map[string][]string
}

// RetryBackoff calculates the duration to wait before the next retry attempt.
type RetryBackoff interface {
	Calculate(attempt int) time.Duration
}

// ExponentialBackoff increases the backoff exponentially: minWait * 2^(attempt-1).
type ExponentialBackoff struct {
	MinWait, MaxWait time.Duration
}

func (b ExponentialBackoff) Calculate(attempt int) time.Duration {
	nextWait := time.Duration(math.Pow(2, float64(attempt-1))) * b.MinWait
	if b.MaxWait > 0 && nextWait > b.MaxWait {
		return b.MaxWait
	}
	return nextWait
}

var timeNow = time.Now

type retryAfterBackoff struct {
	Response *http.Response
	Fallback RetryBackoff
}

func (b retryAfterBackoff) Calculate(attempt int) (waitTime time.Duration) {
	defer func() {
		const maxRetryAfterWaitTime = 5 * time.Minute
		if waitTime < 0 {
			waitTime = b.Fallback.Calculate(attempt)
		} else if waitTime > maxRetryAfterWaitTime {
			waitTime = maxRetryAfterWaitTime
		}
	}()

	// Parse the Retry-After header from a response.
	// It supports both delay-seconds and HTTP-date formats (RFC 7231 §7.1.3).

	header := b.Response.Header.Get("Retry-After")
	if header == "" {
		return -1
	}

	// Try as delay-seconds first.
	if seconds, err := strconv.ParseInt(header, 10, 64); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// Try as HTTP-date (RFC 7231).
	if date, err := http.ParseTime(header); err == nil {
		return date.Sub(timeNow())
	}
	return -1
}

// retryRoundTripper wraps an http.RoundTripper to retry failed requests.
// See WithRetry for which methods are retried.
type retryRoundTripper struct {
	Next                http.RoundTripper
	MaxRetries          int
	ShouldRetryRequest  func(req *http.Request) bool
	ShouldRetryResponse func(resp *http.Response, err error) RetryBackoff
}

func (r *retryRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if !r.ShouldRetryRequest(req) {
		return r.Next.RoundTrip(req)
	}
	req = makeRequestBodyRetryable(req)
	for attempt := 1; ; attempt++ {
		resp, err := r.Next.RoundTrip(req)
		if errors.Is(err, errRetryableBodyClose) {
			return resp, err
		}
		backoff := r.ShouldRetryResponse(resp, err)
		// No retry needed or no more retries left — return as-is.
		if backoff == nil || attempt > r.MaxRetries {
			return resp, err
		}
		drainAndCloseResponseBody(req.Context(), resp)
		if req.GetBody != nil {
			if body, err := req.GetBody(); err != nil {
				return nil, err
			} else {
				req.Body = body
			}
		}
		waitTime := backoff.Calculate(attempt)
		Log.Warn(req.Context(), "retrying request", append(
			func() []any {
				if err != nil {
					return []any{"error", err.Error()}
				}
				return []any{"status", resp.StatusCode}
			}(),
			"method", req.Method,
			"path", req.URL.Path,
			"attempt", fmt.Sprintf("%d/%d", attempt, r.MaxRetries),
			"waitTime", waitTime,
		)...)
		timer := time.NewTimer(waitTime)
		select {
		case <-req.Context().Done():
			timer.Stop()
			return nil, req.Context().Err()
		case <-timer.C:
		}
	}
}

func makeRequestBodyRetryable(req *http.Request) *http.Request {
	if req.Body == nil {
		return req
	}
	// If GetBody already returns independent readers (e.g. set by http.NewRequestWithContext
	// for *bytes.Buffer, *bytes.Reader, *strings.Reader), use it as-is for retries.
	if req.GetBody != nil {
		return req
	}
	body := retryableBody{Closer: req.Body}
	body.Reader = io.TeeReader(req.Body, &body.Buffer)
	result := req.Clone(req.Context())
	result.Body = &body
	result.GetBody = nil
	return result
}

// retryableBody lazily captures request body bytes on the first read and replays them on retries.
// Buffer is filled via TeeReader as the transport reads during the first request. On Close, the
// source is released and subsequent reads replay from Buffer via bytes.NewReader.
type retryableBody struct {
	io.Reader
	io.Closer
	Buffer appendWriter
}

var errRetryableBodyClose = errors.New("retryableBody failed to close")

func (b *retryableBody) Close() error {
	// Drain remaining bytes through the TeeReader to ensure Buffer captures the full body,
	// even if the transport only partially read it (e.g. connection reset mid-write).
	if _, err := io.Copy(io.Discard, b.Reader); err != nil {
		return errors.Join(err, errRetryableBodyClose)
	}
	// On first close, close the Body and use the b.Buffer from now on
	if b.Closer != nil {
		if err := b.Closer.Close(); err != nil {
			return errors.Join(err, errRetryableBodyClose)
		}
	}
	b.Closer = nil
	b.Reader = bytes.NewReader(b.Buffer)
	return nil
}

// appendWriter is an io.Writer that appends to a []byte slice.
// Helper for retryableBody.Buffer.
type appendWriter []byte

func (w *appendWriter) Write(p []byte) (int, error) {
	*w = append(*w, p...)
	return len(p), nil
}

// drainAndCloseResponseBody reads up to maxBytes from the response body before closing it.
// Draining enables Go's http.Transport to reuse the underlying TCP connection for
// subsequent requests. The maxBytes limit prevents getting stuck on large or slow
// responses — if the body exceeds this limit, the connection won't be reused, but
// we won't block indefinitely either.
func drainAndCloseResponseBody(ctx context.Context, resp *http.Response) {
	const maxBytes = 16 * 1024
	if resp != nil && resp.Body != nil {
		drainedBytes, err := io.CopyN(io.Discard, resp.Body, maxBytes)
		if err != nil && !errors.Is(err, io.EOF) {
			Log.Debug(ctx, fmt.Sprintf("failed to drain response body: %s", err.Error()))
		}
		if err := resp.Body.Close(); err != nil {
			Log.Debug(ctx, fmt.Sprintf("failed to close response body after draining %d bytes: %s", drainedBytes, err.Error()))
		}
	}
}
