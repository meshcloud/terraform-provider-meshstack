package internal

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Authorization interface {
	Header(ctx context.Context, client HttpClient) (string, error)
}

func NewClientSecretAuthorization(loginApiPath, clientId, clientSecret string) Authorization {
	return &clientSecretAuthorization{
		LoginApiPath: loginApiPath,
		ClientId:     clientId,
		ClientSecret: clientSecret,
	}
}

type BearerTokenAuthorization struct {
	Token string
}

func (auth BearerTokenAuthorization) Header(_ context.Context, _ HttpClient) (string, error) {
	return fmt.Sprintf("Bearer %s", auth.Token), nil
}

type clientSecretAuthorization struct {
	BearerTokenAuthorization
	LoginApiPath string
	ClientId     string
	ClientSecret string
	ExpiresAt    time.Time
	mu           sync.Mutex
}

func (auth *clientSecretAuthorization) Header(ctx context.Context, client HttpClient) (string, error) {
	auth.mu.Lock()
	defer auth.mu.Unlock()
	if err := auth.ensureValidToken(ctx, client); err != nil {
		return "", err
	}
	return auth.BearerTokenAuthorization.Header(ctx, client)
}

func (auth *clientSecretAuthorization) ensureValidToken(ctx context.Context, client HttpClient) error {
	const minimumTokenLifetime = 30 * time.Second
	if auth.Token != "" && time.Until(auth.ExpiresAt) > minimumTokenLifetime {
		return nil
	}

	loginApiUrl := client.RootUrl.JoinPath(auth.LoginApiPath)

	type loginRequest struct {
		ClientId     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
	}

	type loginResponse struct {
		Token     string `json:"access_token"`
		ExpireSec int    `json:"expires_in"`
	}

	loginResult, err := DoRequest[loginResponse](ctx, client, http.MethodPost, loginApiUrl,
		withPayload(loginRequest{ClientId: auth.ClientId, ClientSecret: auth.ClientSecret}, "application/json"),
	)
	if err != nil {
		return fmt.Errorf("login at %s with client id '%s' failed: %w", loginApiUrl, auth.ClientId, err)
	}
	auth.Token = loginResult.Token
	auth.ExpiresAt = time.Now().Add(time.Duration(loginResult.ExpireSec) * time.Second)
	Log.Debug(ctx, "login successful", "url", loginApiUrl, "clientId", auth.ClientId, "expiresAt", auth.ExpiresAt)
	return nil
}
