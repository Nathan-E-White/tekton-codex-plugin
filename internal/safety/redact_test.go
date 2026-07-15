package safety_test

import (
	"strings"
	"testing"

	"github.com/Nathan-E-White/tekton-codex-plugin/internal/safety"
)

func TestRedactRemovesCommonCredentialShapes(t *testing.T) {
	in := "Authorization: Bearer abc.def.ghi\npassword=hunter2\ntoken: ghp_1234567890abcdef\n-----BEGIN PRIVATE KEY-----\nraw\n-----END PRIVATE KEY-----"
	got := safety.Redact(in)
	for _, secret := range []string{"abc.def.ghi", "hunter2", "ghp_1234567890abcdef", "raw"} {
		if strings.Contains(got, secret) {
			t.Fatalf("Redact() retained %q in %q", secret, got)
		}
	}
	if !strings.Contains(got, "[REDACTED]") {
		t.Fatalf("Redact() = %q, want redaction marker", got)
	}
}
