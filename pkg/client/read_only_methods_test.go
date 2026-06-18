package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newReadOnlyMockServer(t *testing.T, responses map[string]string, seen *[]string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		*seen = append(*seen, path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if body, ok := responses[path]; ok {
			if _, err := w.Write([]byte(body)); err != nil {
				t.Fatalf("failed to write mock response: %v", err)
			}
			return
		}
		if _, err := w.Write([]byte(`{"error":404,"message":"Endpoint not found"}`)); err != nil {
			t.Fatalf("failed to write mock response: %v", err)
		}
	}))
}

func TestClient_GetSuspensionDetails_Mock(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		wantRecords  int
		wantEvidence string
	}{
		{
			name: "no records",
			body: `{
				"error": 0,
				"suspension_count": 0,
				"total_abuse_points": 0,
				"max_abuse_points": 60
			}`,
			wantRecords: 0,
		},
		{
			name: "with records",
			body: `{
				"error": 0,
				"suspension_count": 1,
				"total_abuse_points": 20,
				"max_abuse_points": 60,
				"suspensions": [
					{
						"record_id": 123,
						"flag": "spam",
						"is_soft": 1,
						"evidence_record_id": 456,
						"abuse_points": 20
					}
				],
				"evidence": {
					"456": "sample evidence"
				}
			}`,
			wantRecords:  1,
			wantEvidence: "sample evidence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seen := []string{}
			server := newReadOnlyMockServer(t, map[string]string{
				"getSuspensionDetails": tt.body,
			}, &seen)
			defer server.Close()

			c := NewClient("valid_key", "123456")
			c.SetBaseURL(server.URL)

			resp, err := c.GetSuspensionDetails(context.Background())
			if err != nil {
				t.Fatalf("GetSuspensionDetails() error = %v", err)
			}

			if len(seen) != 1 || seen[0] != "getSuspensionDetails" {
				t.Fatalf("endpoint = %v, want [getSuspensionDetails]", seen)
			}
			if len(resp.Suspensions) != tt.wantRecords {
				t.Fatalf("suspensions length = %d, want %d", len(resp.Suspensions), tt.wantRecords)
			}
			if tt.wantEvidence != "" && resp.Evidence["456"] != tt.wantEvidence {
				t.Errorf("evidence = %q, want %q", resp.Evidence["456"], tt.wantEvidence)
			}
		})
	}
}

func TestClient_GetPolicyViolations_Mock(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantRecords int
	}{
		{
			name: "no records",
			body: `{
				"error": 0,
				"total_abuse_points": 0,
				"max_abuse_points": 60
			}`,
			wantRecords: 0,
		},
		{
			name: "with records",
			body: `{
				"error": 0,
				"total_abuse_points": 10,
				"max_abuse_points": 60,
				"policy_violations": [
					{
						"record_id": 789,
						"timestamp": 1710000000,
						"suspend_at": 1710003600,
						"flag": "policy",
						"is_soft": 0,
						"abuse_points": 10,
						"evidence_data": "sample policy evidence"
					}
				]
			}`,
			wantRecords: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seen := []string{}
			server := newReadOnlyMockServer(t, map[string]string{
				"getPolicyViolations": tt.body,
			}, &seen)
			defer server.Close()

			c := NewClient("valid_key", "123456")
			c.SetBaseURL(server.URL)

			resp, err := c.GetPolicyViolations(context.Background())
			if err != nil {
				t.Fatalf("GetPolicyViolations() error = %v", err)
			}

			if len(seen) != 1 || seen[0] != "getPolicyViolations" {
				t.Fatalf("endpoint = %v, want [getPolicyViolations]", seen)
			}
			if len(resp.PolicyViolations) != tt.wantRecords {
				t.Fatalf("policy_violations length = %d, want %d", len(resp.PolicyViolations), tt.wantRecords)
			}
		})
	}
}

func TestClient_GetNotificationPreferences_Mock(t *testing.T) {
	seen := []string{}
	server := newReadOnlyMockServer(t, map[string]string{
		"kiwivm/getNotificationPreferences": `{
			"error": 0,
			"notificationEmail": "user@example.com",
			"email_preferences": {
				"service": {
					"maintenance": {
						"friendly_description": "Maintenance notices",
						"is_enabled": 1,
						"changed_timestamp": 1710000000,
						"s_value": "daily"
					}
				}
			}
		}`,
	}, &seen)
	defer server.Close()

	c := NewClient("valid_key", "123456")
	c.SetBaseURL(server.URL)

	resp, err := c.GetNotificationPreferences(context.Background())
	if err != nil {
		t.Fatalf("GetNotificationPreferences() error = %v", err)
	}

	if len(seen) != 1 || seen[0] != "kiwivm/getNotificationPreferences" {
		t.Fatalf("endpoint = %v, want [kiwivm/getNotificationPreferences]", seen)
	}
	if resp.NotificationEmail != "user@example.com" {
		t.Errorf("NotificationEmail = %q, want user@example.com", resp.NotificationEmail)
	}
	pref := resp.EmailPreferences["service"]["maintenance"]
	if pref.FriendlyDescription != "Maintenance notices" {
		t.Errorf("FriendlyDescription = %q, want Maintenance notices", pref.FriendlyDescription)
	}
	if pref.IsEnabled != 1 {
		t.Errorf("IsEnabled = %d, want 1", pref.IsEnabled)
	}
	if pref.ChangedTimestamp != 1710000000 {
		t.Errorf("ChangedTimestamp = %d, want 1710000000", pref.ChangedTimestamp)
	}
	if pref.SValue != "daily" {
		t.Errorf("SValue = %q, want daily", pref.SValue)
	}
}

func TestClient_NewReadOnlyMethods_BWHError(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		call     func(context.Context, *Client) error
	}{
		{
			name:     "suspension details",
			endpoint: "getSuspensionDetails",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.GetSuspensionDetails(ctx)
				return err
			},
		},
		{
			name:     "policy violations",
			endpoint: "getPolicyViolations",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.GetPolicyViolations(ctx)
				return err
			},
		},
		{
			name:     "notification preferences",
			endpoint: "kiwivm/getNotificationPreferences",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.GetNotificationPreferences(ctx)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seen := []string{}
			server := newReadOnlyMockServer(t, map[string]string{
				tt.endpoint: `{"error":700005,"message":"Authentication failure"}`,
			}, &seen)
			defer server.Close()

			c := NewClient("invalid_key", "123456")
			c.SetBaseURL(server.URL)

			err := tt.call(context.Background(), c)
			if err == nil {
				t.Fatal("expected BWH error")
			}
			if len(seen) != 1 || seen[0] != tt.endpoint {
				t.Fatalf("endpoint = %v, want [%s]", seen, tt.endpoint)
			}
			if !IsBWHError(err) {
				t.Fatalf("expected BWHError, got %T: %v", err, err)
			}
			bwhErr, ok := GetBWHError(err)
			if !ok {
				t.Fatal("failed to extract BWHError")
			}
			if bwhErr.Code != 700005 {
				t.Errorf("BWHError.Code = %d, want 700005", bwhErr.Code)
			}
		})
	}
}
