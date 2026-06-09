// Package audit implements the `me audit` / `mine audit` command, which lists
// the current user's Jira activity for a given day by reconstructing it from
// issue changelogs (Search API with expand=changelog).
package audit

import (
	"fmt"
	"strings"
	"time"

	"AndersSpringborg/jira-cli/internal/cmdutil"
	"AndersSpringborg/jira-cli/internal/output"

	"github.com/spf13/cobra"
)

// jiraTimeLayout is the timestamp format Jira uses in changelog entries.
const jiraTimeLayout = "2006-01-02T15:04:05.000-0700"

// NewCmd creates the `audit` subcommand. It is mounted under both `me` and
// `mine` so `jira me audit` and `jira mine audit` are equivalent.
func NewCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		date       string
		maxResults int
		columns    string
		raw        bool
	)

	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Show your Jira activity for a given day",
		Long: `List your own changes to Jira issues on a given day.

Reconstructs your activity from issue changelogs: finds issues you updated in
the day window, then reports each field/status change you authored.

Examples:
  jira me audit
  jira me audit --date 2026-06-07
  jira mine audit --date 2026-06-07 --format markdown`,
		RunE: func(cmd *cobra.Command, args []string) error {
			day, err := resolveDay(date)
			if err != nil {
				return err
			}

			client, err := f.LoadClient()
			if err != nil {
				return err
			}

			me, err := client.GetMyself()
			if err != nil {
				return err
			}
			userID := currentUserID(me)
			if userID == "" {
				return fmt.Errorf("could not determine current user id from /myself")
			}

			from := day.Format("2006-01-02")
			to := day.AddDate(0, 0, 1).Format("2006-01-02")
			jql := fmt.Sprintf(`issuekey IN updatedBy(%q, %q, %q)`, userID, from, to)

			data, err := client.SearchWithChangelog(jql, maxResults)
			if err != nil {
				return err
			}

			driver := f.DisplayDriver(cmd)
			if raw {
				return driver.Raw(data)
			}

			issues, ok := data["issues"].([]any)
			if !ok || len(issues) == 0 {
				return driver.Message("No activity found for %s.", from)
			}

			rows := collectMyChanges(issues, userID, day)
			if len(rows) == 0 {
				return driver.Message("No activity found for %s.", from)
			}

			cols := output.NormalizeFields(columns, []string{"time", "key", "field", "from", "to"})
			return driver.List("Audit", cols, rows)
		},
	}

	cmd.Flags().StringVar(&date, "date", "", "Day to audit (YYYY-MM-DD, default today)")
	cmd.Flags().IntVar(&maxResults, "max", 50, "Max issues to scan")
	cmd.Flags().StringVar(&columns, "columns", "", "Comma-separated columns to display")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON response")

	return cmd
}

// resolveDay parses the --date flag (YYYY-MM-DD) or defaults to today (local).
func resolveDay(date string) (time.Time, error) {
	if date == "" {
		now := time.Now()
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()), nil
	}
	day, err := time.ParseInLocation("2006-01-02", date, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid --date %q, expected YYYY-MM-DD", date)
	}
	return day, nil
}

// currentUserID extracts the identifier used by the updatedBy() JQL function:
// accountId on Cloud, falling back to name/key on Server/DC.
func currentUserID(me map[string]any) string {
	for _, k := range []string{"accountId", "name", "key"} {
		if v, ok := me[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// collectMyChanges walks each issue's changelog and returns one row per change
// item authored by userID on the target day. Pure (no I/O) for testability.
func collectMyChanges(issues []any, userID string, day time.Time) []map[string]any {
	var rows []map[string]any

	for _, raw := range issues {
		iss, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		key, _ := iss["key"].(string)
		var summary any
		if flds, ok := iss["fields"].(map[string]any); ok {
			summary = flds["summary"]
		}

		changelog, ok := iss["changelog"].(map[string]any)
		if !ok {
			continue
		}
		histories, ok := changelog["histories"].([]any)
		if !ok {
			continue
		}

		for _, h := range histories {
			hist, ok := h.(map[string]any)
			if !ok {
				continue
			}
			if !authorMatches(hist["author"], userID) {
				continue
			}
			created, ok := hist["created"].(string)
			if !ok || !sameDay(created, day) {
				continue
			}
			items, ok := hist["items"].([]any)
			if !ok {
				continue
			}
			for _, it := range items {
				item, ok := it.(map[string]any)
				if !ok {
					continue
				}
				rows = append(rows, map[string]any{
					"time":    created,
					"key":     key,
					"summary": summary,
					"field":   item["field"],
					"from":    item["fromString"],
					"to":      item["toString"],
				})
			}
		}
	}

	return rows
}

// authorMatches reports whether the changelog author is the current user,
// matching on accountId (Cloud) or name/key (Server/DC).
func authorMatches(author any, userID string) bool {
	a, ok := author.(map[string]any)
	if !ok {
		return false
	}
	for _, k := range []string{"accountId", "name", "key"} {
		if v, ok := a[k].(string); ok && v == userID {
			return true
		}
	}
	return false
}

// sameDay reports whether a Jira timestamp falls on the target day, compared in
// the target day's location.
func sameDay(created string, day time.Time) bool {
	t, err := time.Parse(jiraTimeLayout, strings.TrimSpace(created))
	if err != nil {
		return false
	}
	t = t.In(day.Location())
	return t.Year() == day.Year() && t.Month() == day.Month() && t.Day() == day.Day()
}
