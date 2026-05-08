package internal

import (
	"fmt"
	"net/http"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExponentialBackoff_Calculate(t *testing.T) {
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{1, 1 * time.Second},
		{2, 2 * time.Second},
		{3, 4 * time.Second},
		{4, 5 * time.Second},
		{5, 5 * time.Second},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt %d", tt.attempt), func(t *testing.T) {
			b := ExponentialBackoff{
				MinWait: 1 * time.Second,
				MaxWait: 5 * time.Second,
			}
			assert.Equalf(t, tt.want, b.Calculate(tt.attempt), "Calculate(%v)", tt.attempt)
		})
	}
}

func TestRetryAfterBackoff(t *testing.T) {
	// synctest bubble starts at 2000-01-01T00:00:00Z
	bubbleStart := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	fallback := ExponentialBackoff{MinWait: 1 * time.Second, MaxWait: 10 * time.Second}

	tests := []struct {
		name   string
		header string
		want   time.Duration
	}{
		{"delay-seconds", "30", 30 * time.Second},
		{"zero seconds", "0", 0},                                        // RFC: retry immediately
		{"capped at 5 minutes", "600", 5 * time.Minute},                 // capped
		{"empty header", "", 1 * time.Second},                           // falls back
		{"unparseable header", "not-a-number-or-date", 1 * time.Second}, // falls back
		{"HTTP-date in the past", bubbleStart.Add(-10 * time.Second).Format(http.TimeFormat), 1 * time.Second}, // falls back
		{"HTTP-date in the future", bubbleStart.Add(45 * time.Second).Format(http.TimeFormat), 45 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				b := retryAfterBackoff{
					Response: &http.Response{Header: http.Header{"Retry-After": {tt.header}}},
					Fallback: fallback,
				}
				assert.Equal(t, tt.want, b.Calculate(1))
			})
		})
	}
}
