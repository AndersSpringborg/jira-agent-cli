package issue

import "AndersSpringborg/jira-cli/internal/output"

func writeMutationResult(driver output.DisplayDriver, data map[string]any) error {
	return driver.Item("Result", data)
}
