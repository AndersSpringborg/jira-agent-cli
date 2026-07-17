package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNPMInstallDelegatesUpdateToNPM(t *testing.T) {
	var name string
	var args []string
	u := updater{
		version:       "1.0.0",
		installMethod: "npm",
		lookPath: func(file string) (string, error) {
			assert.Equal(t, "npm", file)
			return "/usr/bin/npm", nil
		},
		run: func(_ context.Context, command string, commandArgs []string, _, _ io.Writer) error {
			name = command
			args = commandArgs
			return nil
		},
	}

	got, err := u.update(context.Background(), io.Discard)
	require.NoError(t, err)
	assert.Equal(t, "/usr/bin/npm", name)
	assert.Equal(t, []string{"install", "-g", "@888aaen/jira-cli@latest"}, args)
	assert.Equal(t, result{Method: "npm", PreviousVersion: "1.0.0", Updated: true}, got)
}

func TestDirectInstallDownloadsVerifiesAndReplacesBinary(t *testing.T) {
	archive := makeArchive(t, "new-binary")
	sum := sha256.Sum256(archive)
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest":
			_, _ = fmt.Fprintf(w, `{"tag_name":"v1.2.0","assets":[{"name":"jira-cli-1.2.0-darwin-arm64.tar.gz","browser_download_url":"%s/archive"},{"name":"checksums.txt","browser_download_url":"%s/checksums"}]}`, serverURL, serverURL)
		case "/archive":
			_, _ = w.Write(archive)
		case "/checksums":
			_, _ = fmt.Fprintf(w, "%x  jira-cli-1.2.0-darwin-arm64.tar.gz\n", sum)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	executable := filepath.Join(t.TempDir(), "jira")
	require.NoError(t, os.WriteFile(executable, []byte("old-binary"), 0o755))
	u := updater{
		version:    "1.0.0",
		latestURL:  server.URL + "/latest",
		client:     server.Client(),
		goos:       "darwin",
		goarch:     "arm64",
		executable: func() (string, error) { return executable, nil },
	}

	got, err := u.update(context.Background(), io.Discard)
	require.NoError(t, err)
	resolvedExecutable, err := filepath.EvalSymlinks(executable)
	require.NoError(t, err)
	assert.Equal(t, result{Method: "github", PreviousVersion: "1.0.0", Version: "1.2.0", Path: resolvedExecutable, Updated: true}, got)
	contents, err := os.ReadFile(executable)
	require.NoError(t, err)
	assert.Equal(t, "new-binary", string(contents))
}

func TestDirectInstallRejectsInvalidChecksumWithoutReplacingBinary(t *testing.T) {
	archive := makeArchive(t, "new-binary")
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest":
			_, _ = fmt.Fprintf(w, `{"tag_name":"v1.2.0","assets":[{"name":"jira-cli-1.2.0-linux-amd64.tar.gz","browser_download_url":"%s/archive"},{"name":"checksums.txt","browser_download_url":"%s/checksums"}]}`, serverURL, serverURL)
		case "/archive":
			_, _ = w.Write(archive)
		case "/checksums":
			_, _ = fmt.Fprintln(w, "deadbeef  jira-cli-1.2.0-linux-amd64.tar.gz")
		}
	}))
	defer server.Close()
	serverURL = server.URL

	executable := filepath.Join(t.TempDir(), "jira")
	require.NoError(t, os.WriteFile(executable, []byte("old-binary"), 0o755))
	u := updater{version: "1.0.0", latestURL: server.URL + "/latest", client: server.Client(), goos: "linux", goarch: "amd64", executable: func() (string, error) { return executable, nil }}

	_, err := u.update(context.Background(), io.Discard)
	require.ErrorContains(t, err, "checksum")
	contents, readErr := os.ReadFile(executable)
	require.NoError(t, readErr)
	assert.Equal(t, "old-binary", string(contents))
}

func TestDirectInstallDoesNothingWhenAlreadyCurrent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"tag_name":"v1.2.0","assets":[]}`)
	}))
	defer server.Close()
	u := updater{version: "1.2.0", latestURL: server.URL, client: server.Client(), goos: "linux", goarch: "amd64"}

	got, err := u.update(context.Background(), io.Discard)
	require.NoError(t, err)
	assert.Equal(t, result{Method: "github", PreviousVersion: "1.2.0", Version: "1.2.0", Updated: false}, got)
}

func TestArchiveNameUsesGoReleaserPlatformNames(t *testing.T) {
	assert.Equal(t, "jira-cli-1.2.3-linux-amd64.tar.gz", archiveName("1.2.3", "linux", "amd64"))
	assert.Equal(t, "jira-cli-1.2.3-windows-amd64.zip", archiveName("1.2.3", "windows", "amd64"))
}

func makeArchive(t *testing.T, binary string) []byte {
	t.Helper()
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	tw := tar.NewWriter(gz)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "jira", Mode: 0o755, Size: int64(len(binary))}))
	_, err := tw.Write([]byte(binary))
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return compressed.Bytes()
}
