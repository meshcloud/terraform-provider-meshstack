package client

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTokenExpiration(t *testing.T) {
	// Helper to create a dummy JWT with a specific expiration time
	createToken := func(expTime time.Time) string {
		header := `{"alg":"HS256","typ":"JWT"}`
		payload := map[string]any{
			"sub":  "1234567890",
			"name": "John Doe",
			"exp":  expTime.Unix(),
		}

		payloadBytes, _ := json.Marshal(payload)

		encodedHeader := base64.RawURLEncoding.EncodeToString([]byte(header))
		encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)
		signature := "dummy_signature"

		return fmt.Sprintf("%s.%s.%s", encodedHeader, encodedPayload, signature)
	}

	fixedBaseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	t.Run("Valid token", func(t *testing.T) {
		expTime := fixedBaseTime
		token := createToken(expTime)

		parsedTime, err := parseTokenExpiration(token)

		require.NoError(t, err)
		assert.Equal(t, expTime.Unix(), parsedTime.Unix())
	})

	t.Run("Invalid format - not enough parts", func(t *testing.T) {
		token := "invalid.token"
		_, err := parseTokenExpiration(token)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid token format")
	})

	t.Run("Invalid Base64 payload", func(t *testing.T) {
		token := "header.invalid_base64$.signature"
		_, err := parseTokenExpiration(token)
		assert.Error(t, err)
	})

	t.Run("Missing exp claim", func(t *testing.T) {
		header := `{"alg":"HS256","typ":"JWT"}`
		payload := `{"sub":"1234567890"}` // No exp
		encodedHeader := base64.RawURLEncoding.EncodeToString([]byte(header))
		encodedPayload := base64.RawURLEncoding.EncodeToString([]byte(payload))
		token := fmt.Sprintf("%s.%s.sig", encodedHeader, encodedPayload)

		_, err := parseTokenExpiration(token)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expiration claim missing")
	})
}
