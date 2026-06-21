package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_WriteAbuseMethods_Mock(t *testing.T) {
	tests := []struct {
		name       string
		endpoint   string
		call       func(context.Context, *Client) error
		wantRecord string
	}{
		{
			name:     "unsuspend",
			endpoint: "unsuspend",
			call: func(ctx context.Context, c *Client) error {
				return c.Unsuspend(ctx, 123)
			},
			wantRecord: "123",
		},
		{
			name:     "resolve policy violation",
			endpoint: "resolvePolicyViolation",
			call: func(ctx context.Context, c *Client) error {
				return c.ResolvePolicyViolation(ctx, 789)
			},
			wantRecord: "789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path := r.URL.Path[1:]
				if path != tt.endpoint {
					t.Fatalf("endpoint = %s, want %s", path, tt.endpoint)
				}
				if got := r.URL.Query().Get("record_id"); got != tt.wantRecord {
					t.Fatalf("record_id = %q, want %q", got, tt.wantRecord)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"error":0}`))
			}))
			defer server.Close()

			c := NewClient("valid_key", "123456")
			c.SetBaseURL(server.URL)

			if err := tt.call(context.Background(), c); err != nil {
				t.Fatalf("%s error = %v", tt.name, err)
			}
		})
	}
}

func TestClient_WriteMethods_InvalidInput(t *testing.T) {
	c := NewClient("valid_key", "123456")

	if err := c.Unsuspend(context.Background(), 0); err == nil {
		t.Fatal("Unsuspend() error = nil, want invalid record_id error")
	}
	if err := c.ResolvePolicyViolation(context.Background(), -1); err == nil {
		t.Fatal("ResolvePolicyViolation() error = nil, want invalid record_id error")
	}
	if _, err := c.SetNotificationPreferences(context.Background(), map[string]bool{}); err == nil {
		t.Fatal("SetNotificationPreferences() error = nil, want empty preferences error")
	}
	if _, err := c.SetNotificationPreferences(context.Background(), map[string]bool{" ": true}); err == nil {
		t.Fatal("SetNotificationPreferences() error = nil, want empty preference id error")
	}
	if _, err := encodeNotificationPreferences(map[string]bool{
		"security-successful-login":   true,
		" security-successful-login ": false,
	}); err == nil {
		t.Fatal("encodeNotificationPreferences() error = nil, want duplicate preference id error")
	}
}

func TestClient_SetNotificationPreferences_Mock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path[1:]
		if path != "kiwivm/setNotificationPreferences" {
			t.Fatalf("endpoint = %s, want kiwivm/setNotificationPreferences", path)
		}

		var sent map[string]int
		if err := json.Unmarshal([]byte(r.URL.Query().Get("json_notification_preferences")), &sent); err != nil {
			t.Fatalf("json_notification_preferences decode error = %v", err)
		}
		if sent["bandwidth-usage-alert-80"] != 1 {
			t.Fatalf("bandwidth-usage-alert-80 = %d, want 1", sent["bandwidth-usage-alert-80"])
		}
		if sent["security-successful-login"] != 0 {
			t.Fatalf("security-successful-login = %d, want 0", sent["security-successful-login"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"error": 0,
			"submitted_email_preferences": {
				"bandwidth-usage-alert-80": 1,
				"security-successful-login": 0
			},
			"updated_email_preferences": {
				"security-successful-login": 0
			},
			"friendly_descriptions": {
				"security-successful-login": "Successful login to KiwiVM"
			}
		}`))
	}))
	defer server.Close()

	c := NewClient("valid_key", "123456")
	c.SetBaseURL(server.URL)

	resp, err := c.SetNotificationPreferences(context.Background(), map[string]bool{
		"bandwidth-usage-alert-80":  true,
		"security-successful-login": false,
	})
	if err != nil {
		t.Fatalf("SetNotificationPreferences() error = %v", err)
	}

	if resp.SubmittedEmailPreferences["bandwidth-usage-alert-80"] != 1 {
		t.Fatalf("submitted state = %d, want 1", resp.SubmittedEmailPreferences["bandwidth-usage-alert-80"])
	}
	if resp.UpdatedEmailPreferences["security-successful-login"] != 0 {
		t.Fatalf("updated state = %d, want 0", resp.UpdatedEmailPreferences["security-successful-login"])
	}
	if resp.FriendlyDescriptions["security-successful-login"] != "Successful login to KiwiVM" {
		t.Fatalf("friendly description = %q", resp.FriendlyDescriptions["security-successful-login"])
	}
}

func TestSetNotificationPreferencesResponse_EmptyArrayMaps(t *testing.T) {
	var resp SetNotificationPreferencesResponse
	err := json.Unmarshal([]byte(`{
		"error": 0,
		"submitted_email_preferences": [],
		"updated_email_preferences": [],
		"friendly_descriptions": []
	}`), &resp)
	if err != nil {
		t.Fatalf("SetNotificationPreferencesResponse unmarshal error = %v", err)
	}

	if len(resp.SubmittedEmailPreferences) != 0 {
		t.Fatalf("submitted length = %d, want 0", len(resp.SubmittedEmailPreferences))
	}
	if len(resp.UpdatedEmailPreferences) != 0 {
		t.Fatalf("updated length = %d, want 0", len(resp.UpdatedEmailPreferences))
	}
	if len(resp.FriendlyDescriptions) != 0 {
		t.Fatalf("friendly descriptions length = %d, want 0", len(resp.FriendlyDescriptions))
	}
}

func TestClient_NewWriteMethods_BWHError(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		call     func(context.Context, *Client) error
	}{
		{
			name:     "unsuspend",
			endpoint: "unsuspend",
			call: func(ctx context.Context, c *Client) error {
				return c.Unsuspend(ctx, 123)
			},
		},
		{
			name:     "resolve policy violation",
			endpoint: "resolvePolicyViolation",
			call: func(ctx context.Context, c *Client) error {
				return c.ResolvePolicyViolation(ctx, 789)
			},
		},
		{
			name:     "set notification preferences",
			endpoint: "kiwivm/setNotificationPreferences",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.SetNotificationPreferences(ctx, map[string]bool{"security-successful-login": true})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path := r.URL.Path[1:]
				if path != tt.endpoint {
					t.Fatalf("endpoint = %s, want %s", path, tt.endpoint)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"error":700005,"message":"Authentication failure"}`))
			}))
			defer server.Close()

			c := NewClient("invalid_key", "123456")
			c.SetBaseURL(server.URL)

			err := tt.call(context.Background(), c)
			if err == nil {
				t.Fatal("expected BWH error")
			}
			if !IsBWHError(err) {
				t.Fatalf("expected BWHError, got %T: %v", err, err)
			}
		})
	}
}
