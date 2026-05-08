package internal

import (
	"context"
	"fmt"
	"time"
)

type Authorization interface {
	Header(ctx context.Context, client HttpClient) (string, error)
}

func NewClientSecretAuthorization(loginApiPath, clientId, clientSecret string) Authorization {
	return &clientSecretAuthorization{
		BearerTokenAuthorization{}, // empty token initially, is refreshed on demand in ensureValidToken
		loginApiPath,
		clientId, clientSecret,
		time.Time{}, // expiry also set in ensureValidToken
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
}

func (auth *clientSecretAuthorization) Header(ctx context.Context, client HttpClient) (string, error) {
	if err := auth.ensureValidToken(ctx, client); err != nil {
		return "", err
	}
	return auth.BearerTokenAuthorization.Header(ctx, client)
}

func (auth *clientSecretAuthorization) ensureValidToken(ctx context.Context, client HttpClient) error {
	if auth.Token != "" && time.Until(auth.ExpiresAt) > 30*time.Second {
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

	loginResult, err := unmarshalBody[loginResponse](client.doRequest(ctx, "POST", loginApiUrl,
		withPayload(loginRequest{ClientId: auth.ClientId, ClientSecret: auth.ClientSecret}, "application/json")),
	)
	if err != nil {
		return fmt.Errorf("login at %s with client id '%s' failed: %w", loginApiUrl, auth.ClientId, err)
	}
	auth.Token = loginResult.Token
	auth.ExpiresAt = time.Now().Add(time.Duration(loginResult.ExpireSec) * time.Second)
	return nil
}
