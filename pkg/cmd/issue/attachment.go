package issue

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"AndersSpringborg/jira-cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func newAttachmentCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attachment",
		Short: "Manage issue attachments",
	}
	cmd.AddCommand(newAttachmentDownloadCmd(f))
	return cmd
}

func newAttachmentDownloadCmd(f *cmdutil.Factory) *cobra.Command {
	var outputPath string

	cmd := &cobra.Command{
		Use:   "download <issue-key> <attachment-id-or-filename>",
		Short: "Download an issue attachment to a file",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueKey := strings.ToUpper(args[0])
			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			issueData, err := client.GetIssue(issueKey, []string{"attachment"})
			if err != nil {
				return err
			}
			attachments := issueAttachments(issueData)
			attachment, err := selectAttachment(attachments, args[1])
			if err != nil {
				return fmt.Errorf("issue %s: %w", issueKey, err)
			}

			contentURL, _ := attachment["content"].(string)
			if contentURL == "" {
				return fmt.Errorf("attachment %v has no content URL", attachment["id"])
			}
			resp, err := client.DownloadAttachment(contentURL)
			if err != nil {
				return fmt.Errorf("download attachment %v: %w", attachment["id"], err)
			}
			defer resp.Body.Close() //nolint:errcheck // body is also closed after copying

			if err := writeAttachmentFile(outputPath, resp.Body); err != nil {
				return fmt.Errorf("write attachment: %w", err)
			}

			absolutePath, err := filepath.Abs(outputPath)
			if err != nil {
				absolutePath = outputPath
			}
			result := map[string]any{
				"issueKey": issueKey,
				"id":       attachment["id"],
				"filename": attachment["filename"],
				"mimeType": attachment["mimeType"],
				"size":     attachment["size"],
				"path":     absolutePath,
			}
			return f.DisplayDriverTo(cmd, cmd.OutOrStdout()).Item("Attachment", result)
		},
	}

	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Destination file path")
	_ = cmd.MarkFlagRequired("output")
	return cmd
}

func issueAttachments(issueData map[string]any) []map[string]any {
	fields, _ := issueData["fields"].(map[string]any)
	items, _ := fields["attachment"].([]any)
	attachments := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if attachment, ok := item.(map[string]any); ok {
			attachments = append(attachments, attachment)
		}
	}
	return attachments
}

func selectAttachment(attachments []map[string]any, selector string) (map[string]any, error) {
	for _, attachment := range attachments {
		if fmt.Sprint(attachment["id"]) == selector {
			return attachment, nil
		}
	}

	var matches []map[string]any
	for _, attachment := range attachments {
		if filename, _ := attachment["filename"].(string); filename == selector {
			matches = append(matches, attachment)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("attachment %q not found", selector)
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("attachment filename %q is ambiguous; use an attachment ID", selector)
	}
}

func writeAttachmentFile(path string, src io.Reader) error {
	if path == "" {
		return fmt.Errorf("output path is required")
	}
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".jira-attachment-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer func() {
		_ = temp.Close()
		_ = os.Remove(tempPath)
	}()

	if _, err := io.Copy(temp, src); err != nil {
		return err
	}
	if err := temp.Sync(); err != nil {
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}
