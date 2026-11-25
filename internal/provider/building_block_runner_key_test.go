package provider

import (
	"context"
	_ "embed"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/meshcloud/meshstack-cli/client"
	"github.com/stretchr/testify/assert"
)

var (
	//go:embed testdata/runner/certificate.pem
	runnerCertificate string
	//go:embed testdata/runner/public.pem
	runnerPublicKeyOfCertificate string
	//go:embed testdata/runner/other-public.pem
	otherRunnerPublicKey string
)

func TestSamePublicKey(t *testing.T) {
	t.Parallel()

	rewrapped := strings.Join(strings.Split(strings.TrimSpace(runnerPublicKeyOfCertificate), "\n"), "\n\n") + "\n"

	assert.True(t, samePublicKey(runnerCertificate, runnerPublicKeyOfCertificate))
	assert.True(t, samePublicKey(runnerPublicKeyOfCertificate, rewrapped))
	assert.True(t, samePublicKey("not a key", "not a key"))
	assert.False(t, samePublicKey(runnerCertificate, otherRunnerPublicKey))
	assert.False(t, samePublicKey(runnerCertificate, "not a key"))
}

type fakeRunnerReader struct {
	client.MeshBuildingBlockRunnerClient
	runners map[string]client.MeshBuildingBlockRunner
	readErr error
}

func (f fakeRunnerReader) Read(_ context.Context, uuid string) (*client.MeshBuildingBlockRunner, error) {
	if f.readErr != nil {
		return nil, f.readErr
	}
	if runner, ok := f.runners[uuid]; ok {
		return &runner, nil
	}
	return nil, nil
}

func TestRunnersShareKey(t *testing.T) {
	t.Parallel()

	runnerWithKey := func(publicKey string) client.MeshBuildingBlockRunner {
		return client.MeshBuildingBlockRunner{Spec: client.MeshBuildingBlockRunnerSpec{PublicKey: publicKey}}
	}
	runners := map[string]client.MeshBuildingBlockRunner{
		"certificate": runnerWithKey(runnerCertificate),
		"bare":        runnerWithKey(runnerPublicKeyOfCertificate),
		"other":       runnerWithKey(otherRunnerPublicKey),
	}

	tests := []struct {
		name          string
		reader        client.MeshBuildingBlockRunnerClient
		uuid, other   string
		wantSharedKey bool
	}{
		{name: "same key", reader: fakeRunnerReader{runners: runners}, uuid: "certificate", other: "bare", wantSharedKey: true},
		{name: "other key", reader: fakeRunnerReader{runners: runners}, uuid: "certificate", other: "other"},
		{name: "runner not found", reader: fakeRunnerReader{runners: runners}, uuid: "certificate", other: "missing"},
		{name: "runner forbidden", reader: fakeRunnerReader{readErr: client.HttpError{StatusCode: http.StatusForbidden}}, uuid: "certificate", other: "bare"},
		{name: "read fails", reader: fakeRunnerReader{readErr: errors.New("connection refused")}, uuid: "certificate", other: "bare"},
		{name: "no runner client", uuid: "certificate", other: "bare"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.wantSharedKey, runnersShareKey(t.Context(), tc.reader, tc.uuid, tc.other))
		})
	}
}
