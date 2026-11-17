// Package registration defines the registration policy shared by the site and CRM.
package registration

import (
	"regexp"
	"strings"
)

var compactPattern = regexp.MustCompile(`^(?:[A-Z]{3}[0-9]{3}|[0-9]{3}[A-Z]{3})$`)
var displayPattern = regexp.MustCompile(`^(?:[A-Z]{3}-[0-9]{3}|[0-9]{3}-[A-Z]{3})$`)

// Normalize preserves order and never truncates or discards unknown characters.
func Normalize(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	compact := strings.NewReplacer(" ", "", "\t", "").Replace(value)
	if compactPattern.MatchString(compact) {
		return compact[:3] + "-" + compact[3:]
	}
	if displayPattern.MatchString(compact) {
		return compact
	}
	return value
}

func Valid(value string) bool { return displayPattern.MatchString(Normalize(value)) }

// Key preserves the existing CRM document/alias IDs (without a hyphen).
func Key(value string) string { return strings.ReplaceAll(Normalize(value), "-", "") }
