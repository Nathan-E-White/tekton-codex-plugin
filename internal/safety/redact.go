package safety

import "regexp"

var redactors = []struct {
	re          *regexp.Regexp
	replacement string
}{
	{regexp.MustCompile(`(?s)-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----.*?-----END [A-Z0-9 ]*PRIVATE KEY-----`), "[REDACTED PRIVATE KEY]"},
	{regexp.MustCompile(`(?i)(authorization:\s*bearer\s+)[^\s]+`), "${1}[REDACTED]"},
	{regexp.MustCompile(`(?i)((?:password|token|secret)\s*[:=]\s*)[^\s]+`), "${1}[REDACTED]"},
	{regexp.MustCompile(`\b(?:ghp|github_pat)_[A-Za-z0-9_]{12,}\b`), "[REDACTED]"},
}

func Redact(value string) string {
	for _, redactor := range redactors {
		value = redactor.re.ReplaceAllString(value, redactor.replacement)
	}
	return value
}
