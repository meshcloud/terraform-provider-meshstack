package provider

import (
	"fmt"
	"net"
	"net/http"
	"net/http/cgi"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// The acceptance suite needs the tf-block-runner to `git clone` the committed bare repo under
// testdata/ and run OpenTofu against it. A file:// URL cannot work in CI: the runner is a separate
// container (its own filesystem), so it never sees the test's testdata/ directory. Both the CI
// runner (ARC kubernetes container mode -> all pod containers share one network namespace, which is
// why the runner reaches meshfed-api at http://127.0.0.1:8300) and a locally `go run` tf-block-runner
// (a host process) CAN reach an http server the test process binds on 127.0.0.1.
//
// So instead of file://, the test serves the bare repo over the git smart-HTTP protocol using the
// system git's git-http-backend via net/http/cgi (pure stdlib + the git binary that is already
// present wherever these tests run). git-http-backend reads the committed loose objects and streams
// a packfile to the client, so the on-disk object encoding (zlib loose objects) is irrelevant to the
// clone -- the repo stays checked in exactly as-is.
var (
	gitHTTPOnce    sync.Once
	gitHTTPBaseURL string
	gitHTTPErr     error
)

// gitHTTPBackendPath locates the git-http-backend CGI helper, which ships with git but lives in
// git's exec-path (e.g. /usr/lib/git-core), not necessarily on PATH.
func gitHTTPBackendPath() (string, error) {
	if out, err := exec.Command("git", "--exec-path").Output(); err == nil {
		p := filepath.Join(strings.TrimSpace(string(out)), "git-http-backend")
		if _, statErr := os.Stat(p); statErr == nil {
			return p, nil
		}
	}
	if p, err := exec.LookPath("git-http-backend"); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("git-http-backend not found in `git --exec-path` or on PATH")
}

// startGitHTTPServer serves testdata/ (the parent of the tf-building-block bare repo) over git
// smart-HTTP on an ephemeral port and returns its base URL (advertised as 127.0.0.1 so the runner
// reaches it across the shared netns). The server runs for the lifetime of the test process.
func startGitHTTPServer() (string, error) {
	projectRoot, err := filepath.Abs("testdata")
	if err != nil {
		return "", err
	}
	backend, err := gitHTTPBackendPath()
	if err != nil {
		return "", err
	}
	// Bind 0.0.0.0 so peers in the pod/host can connect; advertise 127.0.0.1 (loopback is shared
	// across containers in the CI pod and is the host loopback for a local `go run` runner).
	ln, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		return "", err
	}
	srv := &http.Server{
		Handler: &cgi.Handler{
			Path: backend,
			Env: []string{
				"GIT_PROJECT_ROOT=" + projectRoot,
				// Export without requiring a git-daemon-export-ok marker in the bare repo.
				"GIT_HTTP_EXPORT_ALL=1",
			},
		},
	}
	go func() { _ = srv.Serve(ln) }()
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		return "", fmt.Errorf("unexpected listener address type %T", ln.Addr())
	}
	return fmt.Sprintf("http://127.0.0.1:%d", addr.Port), nil
}

// gitHTTPRepoBaseURL lazily starts the smart-HTTP server (once per test process) and returns its
// base URL. It skips the calling test if git-http-backend is unavailable, so unit-only environments
// without a full git install still pass.
func gitHTTPRepoBaseURL(t *testing.T) string {
	t.Helper()
	gitHTTPOnce.Do(func() { gitHTTPBaseURL, gitHTTPErr = startGitHTTPServer() })
	if gitHTTPErr != nil {
		t.Skipf("git smart-HTTP fixture server unavailable: %v", gitHTTPErr)
	}
	return gitHTTPBaseURL
}

// TestBuildingBlockModuleRepoIsCloneable is a self-contained guard (runs in both unit and acceptance
// mode, no backend needed) that the committed bare repo is actually clonable over the smart-HTTP
// transport the runner uses, and that a clone yields the module's main.tf. It catches a corrupted or
// non-bare fixture, or a broken server, without needing the full acceptance stack. The runner uses
// go-git, which speaks the same smart-HTTP protocol; cloning here with the git binary exercises that
// same endpoint.
func TestBuildingBlockModuleRepoIsCloneable(t *testing.T) {
	t.Parallel()

	base := gitHTTPRepoBaseURL(t)
	dst := t.TempDir()
	cmd := exec.Command("git", "clone", "--quiet", base+"/tf-building-block", dst)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git clone over smart-HTTP failed: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(dst, "main.tf")); err != nil {
		t.Fatalf("cloned module missing main.tf: %v", err)
	}
}
