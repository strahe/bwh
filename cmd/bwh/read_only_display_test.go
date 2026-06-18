package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/strahe/bwh/pkg/client"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	os.Stdout = w

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close stdout writer: %v", err)
	}
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read stdout: %v", err)
	}
	return buf.String()
}

func TestDisplaySuspensionDetails(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		out := captureStdout(t, func() {
			displaySuspensionDetails(&client.SuspensionDetailsResponse{
				SuspensionCount:  0,
				TotalAbusePoints: 0,
				MaxAbusePoints:   60,
			})
		})

		if !strings.Contains(out, "No active suspension issues found.") {
			t.Fatalf("output = %q", out)
		}
	})

	t.Run("with records", func(t *testing.T) {
		out := captureStdout(t, func() {
			displaySuspensionDetails(&client.SuspensionDetailsResponse{
				SuspensionCount:  1,
				TotalAbusePoints: 20,
				MaxAbusePoints:   60,
				Suspensions: []client.SuspensionRecord{
					{
						RecordID:         123,
						Flag:             "spam",
						IsSoft:           1,
						EvidenceRecordID: 456,
						AbusePoints:      20,
					},
				},
				Evidence: map[string]string{"456": strings.Repeat("x", 140)},
			})
		})

		for _, want := range []string{"Outstanding Issues (1):", "Case #123", "Soft Resolve:", strings.Repeat("x", 120) + "..."} {
			if !strings.Contains(out, want) {
				t.Fatalf("output missing %q:\n%s", want, out)
			}
		}
	})
}

func TestDisplayPolicyViolations(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		out := captureStdout(t, func() {
			displayPolicyViolations(&client.PolicyViolationsResponse{
				TotalAbusePoints: 0,
				MaxAbusePoints:   60,
			})
		})

		if !strings.Contains(out, "No active policy violations found.") {
			t.Fatalf("output = %q", out)
		}
	})

	t.Run("with records", func(t *testing.T) {
		out := captureStdout(t, func() {
			displayPolicyViolations(&client.PolicyViolationsResponse{
				TotalAbusePoints: 10,
				MaxAbusePoints:   60,
				PolicyViolations: []client.PolicyViolationRecord{
					{
						RecordID:     789,
						Timestamp:    1710000000,
						SuspendAt:    1710003600,
						Flag:         "policy",
						AbusePoints:  10,
						EvidenceData: "sample policy evidence",
					},
				},
			})
		})

		for _, want := range []string{"Active Violations (1):", "Case #789", "sample policy evidence"} {
			if !strings.Contains(out, want) {
				t.Fatalf("output missing %q:\n%s", want, out)
			}
		}
	})
}

func TestDisplayNotificationPreferences(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		out := captureStdout(t, func() {
			displayNotificationPreferences(&client.NotificationPreferencesResponse{})
		})

		if !strings.Contains(out, "No notification preferences found.") {
			t.Fatalf("output = %q", out)
		}
	})

	t.Run("with records", func(t *testing.T) {
		out := captureStdout(t, func() {
			displayNotificationPreferences(&client.NotificationPreferencesResponse{
				NotificationEmail: "user@example.com",
				EmailPreferences: map[string]map[string]client.NotificationPreference{
					"service": {
						"maintenance": {
							FriendlyDescription: "Maintenance notices",
							IsEnabled:           1,
							ChangedTimestamp:    1710000000,
							SValue:              "daily",
						},
					},
				},
			})
		})

		for _, want := range []string{"Email: user@example.com", "maintenance", "Maintenance notices", "Value      : daily"} {
			if !strings.Contains(out, want) {
				t.Fatalf("output missing %q:\n%s", want, out)
			}
		}
	})
}

func TestDisplayDetailedInfoNullroutes(t *testing.T) {
	out := captureStdout(t, func() {
		displayDetailedInfo(&client.LiveServiceInfo{
			ServiceInfo: client.ServiceInfo{
				Hostname:       "test-host",
				VMType:         "kvm",
				Plan:           "test-plan",
				OS:             "debian",
				IPAddresses:    []string{"198.51.100.10"},
				PlanMaxIPv6s:   1,
				MaxAbusePoints: 60,
				IPNullroutes: client.IPNullroutes{
					"192.0.2.20": {},
					"192.0.2.10": {},
				},
			},
		}, "test")
	})

	if !strings.Contains(out, "DDoS Protection  : 2 IP(s) currently null-routed") {
		t.Fatalf("output = %q", out)
	}
	if !strings.Contains(out, "Null-routed IPs  : 192.0.2.10, 192.0.2.20") {
		t.Fatalf("output = %q", out)
	}
}

func TestSummarizeTextUsesRunes(t *testing.T) {
	got := summarizeText("证据内容abc", 4)
	if got != "证据内容..." {
		t.Fatalf("summarizeText() = %q", got)
	}
}
