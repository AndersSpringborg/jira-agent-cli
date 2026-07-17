// Package update provides the self-update command.
package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"AndersSpringborg/jira-cli/internal/output"

	"github.com/spf13/cobra"
)

const (
	defaultLatestURL = "https://api.github.com/repos/AndersSpringborg/jira-agent-cli/releases/latest"
	maxMetadataSize  = 2 << 20
	maxArchiveSize   = 512 << 20
	maxBinarySize    = 256 << 20
)

type result struct {
	Method          string `json:"method"`
	PreviousVersion string `json:"previousVersion"`
	Version         string `json:"version,omitempty"`
	Path            string `json:"path,omitempty"`
	Updated         bool   `json:"updated"`
}

type asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type release struct {
	TagName string  `json:"tag_name"`
	Assets  []asset `json:"assets"`
}

type updater struct {
	version       string
	installMethod string
	latestURL     string
	client        *http.Client
	goos          string
	goarch        string
	executable    func() (string, error)
	lookPath      func(string) (string, error)
	run           func(context.Context, string, []string, io.Writer, io.Writer) error
}

// NewCmd creates the update command.
func NewCmd(version string) *cobra.Command {
	u := updater{
		version:       version,
		installMethod: os.Getenv("JIRA_CLI_INSTALL_METHOD"),
		latestURL:     defaultLatestURL,
		client:        &http.Client{Timeout: 5 * time.Minute},
		goos:          runtime.GOOS,
		goarch:        runtime.GOARCH,
		executable:    os.Executable,
		lookPath:      exec.LookPath,
		run: func(ctx context.Context, name string, args []string, stdout, stderr io.Writer) error {
			command := exec.CommandContext(ctx, name, args...)
			command.Stdin = os.Stdin
			command.Stdout = stdout
			command.Stderr = stderr
			return command.Run()
		},
	}

	return &cobra.Command{
		Use:   "update",
		Short: "Update jira to the latest release",
		Long: `Update jira to the latest published version.

npm installations are upgraded through npm so package metadata and the
platform-specific package remain consistent. Other installations download the
matching GitHub release archive, verify checksums.txt, and replace this binary.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			got, err := u.update(cmd.Context(), cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			formatName, err := cmd.Flags().GetString("format")
			if err != nil {
				return err
			}
			format, err := output.ParseFormat(formatName)
			if err != nil {
				return err
			}
			return output.NewDriverWithWriter(format, cmd.OutOrStdout()).Raw(got)
		},
	}
}

func (u *updater) update(ctx context.Context, progress io.Writer) (result, error) {
	if u.installMethod == "npm" {
		return u.updateWithNPM(ctx, progress)
	}
	return u.updateFromGitHub(ctx)
}

func (u *updater) updateWithNPM(ctx context.Context, progress io.Writer) (result, error) {
	npm, err := u.lookPath("npm")
	if err != nil {
		return result{}, fmt.Errorf("npm installation detected, but npm was not found: %w", err)
	}
	if err := u.run(ctx, npm, []string{"install", "-g", "@888aaen/jira-cli@latest"}, progress, progress); err != nil {
		return result{}, fmt.Errorf("update with npm: %w", err)
	}
	return result{Method: "npm", PreviousVersion: u.version, Updated: true}, nil
}

func (u *updater) updateFromGitHub(ctx context.Context) (result, error) {
	if u.goos == "windows" {
		return result{}, errors.New("direct self-update is not supported on Windows; reinstall with npm install -g @888aaen/jira-cli@latest")
	}

	latest, err := u.latestRelease(ctx)
	if err != nil {
		return result{}, err
	}
	version := strings.TrimPrefix(latest.TagName, "v")
	if version == "" {
		return result{}, errors.New("latest GitHub release has an empty version")
	}
	baseResult := result{Method: "github", PreviousVersion: u.version, Version: version}
	if strings.TrimPrefix(u.version, "v") == version {
		return baseResult, nil
	}

	name := archiveName(version, u.goos, u.goarch)
	archiveURL, err := findAsset(latest.Assets, name)
	if err != nil {
		return result{}, err
	}
	checksumsURL, err := findAsset(latest.Assets, "checksums.txt")
	if err != nil {
		return result{}, err
	}
	archive, err := u.download(ctx, archiveURL, maxArchiveSize)
	if err != nil {
		return result{}, fmt.Errorf("download %s: %w", name, err)
	}
	checksums, err := u.download(ctx, checksumsURL, maxMetadataSize)
	if err != nil {
		return result{}, fmt.Errorf("download checksums.txt: %w", err)
	}
	if err := verifyChecksum(name, archive, checksums); err != nil {
		return result{}, err
	}
	binary, err := extractBinary(archive)
	if err != nil {
		return result{}, fmt.Errorf("extract %s: %w", name, err)
	}

	executable, err := u.executable()
	if err != nil {
		return result{}, fmt.Errorf("resolve executable: %w", err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return result{}, fmt.Errorf("resolve executable symlinks: %w", err)
	}
	if err := replaceExecutable(executable, binary); err != nil {
		return result{}, err
	}
	baseResult.Path = executable
	baseResult.Updated = true
	return baseResult, nil
}

func (u *updater) latestRelease(ctx context.Context) (release, error) {
	body, err := u.download(ctx, u.latestURL, maxMetadataSize)
	if err != nil {
		return release{}, fmt.Errorf("get latest GitHub release: %w", err)
	}
	var latest release
	if err := json.Unmarshal(body, &latest); err != nil {
		return release{}, fmt.Errorf("decode latest GitHub release: %w", err)
	}
	return latest, nil
}

func (u *updater) download(ctx context.Context, url string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "jira-cli/"+u.version)
	resp, err := u.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("response exceeds %d bytes", limit)
	}
	return data, nil
}

func archiveName(version, goos, goarch string) string {
	extension := ".tar.gz"
	if goos == "windows" {
		extension = ".zip"
	}
	return fmt.Sprintf("jira-cli-%s-%s-%s%s", version, goos, goarch, extension)
}

func findAsset(assets []asset, name string) (string, error) {
	for _, candidate := range assets {
		if candidate.Name == name && candidate.URL != "" {
			return candidate.URL, nil
		}
	}
	return "", fmt.Errorf("latest GitHub release does not contain %s", name)
}

func verifyChecksum(name string, archive, checksums []byte) error {
	var expected string
	for _, line := range strings.Split(string(checksums), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.TrimPrefix(fields[len(fields)-1], "*") == name {
			expected = fields[0]
			break
		}
	}
	if expected == "" {
		return fmt.Errorf("checksums.txt does not contain %s", name)
	}
	if _, err := hex.DecodeString(expected); err != nil || len(expected) != sha256.Size*2 {
		return fmt.Errorf("invalid checksum for %s", name)
	}
	actual := sha256.Sum256(archive)
	if !strings.EqualFold(expected, hex.EncodeToString(actual[:])) {
		return fmt.Errorf("checksum mismatch for %s", name)
	}
	return nil
}

func extractBinary(archive []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	defer func() { _ = gz.Close() }()
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if header.Typeflag != tar.TypeReg || filepath.Base(header.Name) != "jira" {
			continue
		}
		if header.Size < 1 || header.Size > maxBinarySize {
			return nil, fmt.Errorf("binary has invalid size %d", header.Size)
		}
		binary, err := io.ReadAll(io.LimitReader(tr, maxBinarySize+1))
		if err != nil {
			return nil, err
		}
		if int64(len(binary)) != header.Size {
			return nil, errors.New("binary is truncated")
		}
		return binary, nil
	}
	return nil, errors.New("archive does not contain jira binary")
}

func replaceExecutable(path string, binary []byte) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".jira-update-*")
	if err != nil {
		return fmt.Errorf("create update beside %s: %w (run the command with permission to write this directory)", path, err)
	}
	tempPath := temp.Name()
	defer func() { _ = os.Remove(tempPath) }()

	if _, err := temp.Write(binary); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write updated binary: %w", err)
	}
	if err := temp.Chmod(0o755); err != nil {
		_ = temp.Close()
		return fmt.Errorf("make updated binary executable: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("sync updated binary: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close updated binary: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}
