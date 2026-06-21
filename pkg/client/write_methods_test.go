package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClient_WriteMethodsUsePostForm(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		call     func(context.Context, *Client) error
		wantForm map[string]string
	}{
		{
			name:     "create snapshot",
			endpoint: "snapshot/create",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.CreateSnapshot(ctx, "backup-name")
				return err
			},
			wantForm: map[string]string{"description": "backup-name"},
		},
		{
			name:     "delete snapshot",
			endpoint: "snapshot/delete",
			call: func(ctx context.Context, c *Client) error {
				return c.DeleteSnapshot(ctx, "snapshot.tar.gz")
			},
			wantForm: map[string]string{"snapshot": "snapshot.tar.gz"},
		},
		{
			name:     "restore snapshot",
			endpoint: "snapshot/restore",
			call: func(ctx context.Context, c *Client) error {
				return c.RestoreSnapshot(ctx, "snapshot.tar.gz")
			},
			wantForm: map[string]string{"snapshot": "snapshot.tar.gz"},
		},
		{
			name:     "toggle snapshot sticky",
			endpoint: "snapshot/toggleSticky",
			call: func(ctx context.Context, c *Client) error {
				return c.ToggleSnapshotSticky(ctx, "snapshot.tar.gz", true)
			},
			wantForm: map[string]string{"snapshot": "snapshot.tar.gz", "sticky": "1"},
		},
		{
			name:     "export snapshot",
			endpoint: "snapshot/export",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.ExportSnapshot(ctx, "snapshot.tar.gz")
				return err
			},
			wantForm: map[string]string{"snapshot": "snapshot.tar.gz"},
		},
		{
			name:     "import snapshot",
			endpoint: "snapshot/import",
			call: func(ctx context.Context, c *Client) error {
				return c.ImportSnapshot(ctx, "654321", "token")
			},
			wantForm: map[string]string{"sourceVeid": "654321", "sourceToken": "token"},
		},
		{
			name:     "restart",
			endpoint: "restart",
			call: func(ctx context.Context, c *Client) error {
				return c.Restart(ctx)
			},
		},
		{
			name:     "start",
			endpoint: "start",
			call: func(ctx context.Context, c *Client) error {
				return c.Start(ctx)
			},
		},
		{
			name:     "stop",
			endpoint: "stop",
			call: func(ctx context.Context, c *Client) error {
				return c.Stop(ctx)
			},
		},
		{
			name:     "kill",
			endpoint: "kill",
			call: func(ctx context.Context, c *Client) error {
				return c.Kill(ctx)
			},
		},
		{
			name:     "reinstall os",
			endpoint: "reinstallOS",
			call: func(ctx context.Context, c *Client) error {
				return c.ReinstallOS(ctx, "debian-12-x86_64")
			},
			wantForm: map[string]string{"os": "debian-12-x86_64"},
		},
		{
			name:     "reset root password",
			endpoint: "resetRootPassword",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.ResetRootPassword(ctx)
				return err
			},
		},
		{
			name:     "copy backup to snapshot",
			endpoint: "backup/copyToSnapshot",
			call: func(ctx context.Context, c *Client) error {
				return c.CopyBackupToSnapshot(ctx, "backup-token")
			},
			wantForm: map[string]string{"backupToken": "backup-token"},
		},
		{
			name:     "set hostname",
			endpoint: "setHostname",
			call: func(ctx context.Context, c *Client) error {
				return c.SetHostname(ctx, "host.example.com")
			},
			wantForm: map[string]string{"newHostname": "host.example.com"},
		},
		{
			name:     "unsuspend",
			endpoint: "unsuspend",
			call: func(ctx context.Context, c *Client) error {
				return c.Unsuspend(ctx, 123)
			},
			wantForm: map[string]string{"record_id": "123"},
		},
		{
			name:     "resolve policy violation",
			endpoint: "resolvePolicyViolation",
			call: func(ctx context.Context, c *Client) error {
				return c.ResolvePolicyViolation(ctx, 789)
			},
			wantForm: map[string]string{"record_id": "789"},
		},
		{
			name:     "set notification preferences",
			endpoint: "kiwivm/setNotificationPreferences",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.SetNotificationPreferences(ctx, map[string]bool{"security-successful-login": true})
				return err
			},
			wantForm: map[string]string{"json_notification_preferences": `{"security-successful-login":1}`},
		},
		{
			name:     "update ssh keys",
			endpoint: "updateSshKeys",
			call: func(ctx context.Context, c *Client) error {
				return c.UpdateSshKeys(ctx, []string{"ssh-rsa key1", "ssh-ed25519 key2"})
			},
			wantForm: map[string]string{"ssh_keys": "ssh-rsa key1\nssh-ed25519 key2\n"},
		},
		{
			name:     "set ptr",
			endpoint: "setPTR",
			call: func(ctx context.Context, c *Client) error {
				return c.SetPTR(ctx, "192.0.2.10", "host.example.com")
			},
			wantForm: map[string]string{"ip": "192.0.2.10", "ptr": "host.example.com"},
		},
		{
			name:     "mount iso",
			endpoint: "iso/mount",
			call: func(ctx context.Context, c *Client) error {
				return c.MountISO(ctx, "ubuntu.iso")
			},
			wantForm: map[string]string{"iso": "ubuntu.iso"},
		},
		{
			name:     "unmount iso",
			endpoint: "iso/unmount",
			call: func(ctx context.Context, c *Client) error {
				return c.UnmountISO(ctx)
			},
		},
		{
			name:     "start migration with timeout",
			endpoint: "migrate/start",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.StartMigrationWithTimeout(ctx, "us-west", time.Second)
				return err
			},
			wantForm: map[string]string{"location": "us-west"},
		},
		{
			name:     "add ipv6",
			endpoint: "ipv6/add",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.AddIPv6(ctx)
				return err
			},
		},
		{
			name:     "delete ipv6",
			endpoint: "ipv6/delete",
			call: func(ctx context.Context, c *Client) error {
				return c.DeleteIPv6(ctx, "2001:db8::/64")
			},
			wantForm: map[string]string{"ip": "2001:db8::/64"},
		},
		{
			name:     "assign private ip",
			endpoint: "privateIp/assign",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.AssignPrivateIP(ctx, "10.0.0.2")
				return err
			},
			wantForm: map[string]string{"ip": "10.0.0.2"},
		},
		{
			name:     "delete private ip",
			endpoint: "privateIp/delete",
			call: func(ctx context.Context, c *Client) error {
				return c.DeletePrivateIP(ctx, "10.0.0.2")
			},
			wantForm: map[string]string{"ip": "10.0.0.2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assertPostForm(t, r, tt.endpoint, "valid_key", tt.wantForm)
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

func TestClient_ReadMethodsUseGetQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if got := r.URL.Query().Get("veid"); got != "123456" {
			t.Fatalf("query veid = %q, want 123456", got)
		}
		if got := r.URL.Query().Get("api_key"); got != "valid_key" {
			t.Fatalf("query api_key = %q, want valid_key", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		if got := r.PostForm.Get("veid"); got != "" {
			t.Fatalf("post form veid = %q, want empty", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":0}`))
	}))
	defer server.Close()

	c := NewClient("valid_key", "123456")
	c.SetBaseURL(server.URL)

	if _, err := c.GetRateLimitStatus(context.Background()); err != nil {
		t.Fatalf("GetRateLimitStatus() error = %v", err)
	}
}

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
				assertPostForm(t, r, tt.endpoint, "valid_key", map[string]string{"record_id": tt.wantRecord})
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
		assertPostForm(t, r, "kiwivm/setNotificationPreferences", "valid_key", nil)

		var sent map[string]int
		if err := json.Unmarshal([]byte(r.PostForm.Get("json_notification_preferences")), &sent); err != nil {
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
				assertPostForm(t, r, tt.endpoint, "invalid_key", nil)
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

func assertPostForm(t *testing.T, r *http.Request, endpoint, apiKey string, want map[string]string) {
	t.Helper()

	if r.Method != http.MethodPost {
		t.Fatalf("method = %s, want POST", r.Method)
	}
	if path := r.URL.Path[1:]; path != endpoint {
		t.Fatalf("endpoint = %s, want %s", path, endpoint)
	}
	if r.URL.RawQuery != "" {
		t.Fatalf("raw query = %q, want empty", r.URL.RawQuery)
	}
	if contentType := r.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		t.Fatalf("Content-Type = %q, want application/x-www-form-urlencoded", contentType)
	}
	if err := r.ParseForm(); err != nil {
		t.Fatalf("ParseForm() error = %v", err)
	}
	if got := r.PostForm.Get("veid"); got != "123456" {
		t.Fatalf("post form veid = %q, want 123456", got)
	}
	if got := r.PostForm.Get("api_key"); got != apiKey {
		t.Fatalf("post form api_key = %q, want %s", got, apiKey)
	}
	for key, wantValue := range want {
		if got := r.PostForm.Get(key); got != wantValue {
			t.Fatalf("post form %s = %q, want %q", key, got, wantValue)
		}
	}
}
