package issue

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectAttachment(t *testing.T) {
	attachments := []map[string]any{
		{"id": "10041", "filename": "report.pdf"},
		{"id": "10042", "filename": "screenshot.png"},
	}

	t.Run("by id", func(t *testing.T) {
		got, err := selectAttachment(attachments, "10042")
		require.NoError(t, err)
		assert.Equal(t, "screenshot.png", got["filename"])
	})

	t.Run("by unique filename", func(t *testing.T) {
		got, err := selectAttachment(attachments, "report.pdf")
		require.NoError(t, err)
		assert.Equal(t, "10041", got["id"])
	})

	t.Run("missing", func(t *testing.T) {
		_, err := selectAttachment(attachments, "missing.txt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("ambiguous filename", func(t *testing.T) {
		duplicates := make([]map[string]any, 0, len(attachments)+1)
		duplicates = append(duplicates, attachments...)
		duplicates = append(duplicates, map[string]any{"id": "10043", "filename": "report.pdf"})
		_, err := selectAttachment(duplicates, "report.pdf")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ambiguous")
	})
}

func TestWriteAttachmentFileIsAtomic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "attachment.bin")
	reader := io.MultiReader(strings.NewReader("partial"), errorReader{})

	err := writeAttachmentFile(path, reader)
	require.Error(t, err)
	_, statErr := os.Stat(path)
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestAttachmentCommandIsRegistered(t *testing.T) {
	cmd := NewCmd(&cmdutil.Factory{})
	download, _, err := cmd.Find([]string{"attachment", "download"})
	require.NoError(t, err)
	assert.Equal(t, "download", download.Name())
	assert.Contains(t, download.Use, "<issue-key> <attachment-id-or-filename>")
	assert.NotNil(t, download.Flags().Lookup("output"))
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}
