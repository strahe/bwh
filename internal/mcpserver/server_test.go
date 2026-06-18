package mcpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/strahe/bwh/internal/config"
	"github.com/strahe/bwh/pkg/client"
)

func newMCPTestManager(t *testing.T, endpoint string) *config.Manager {
	t.Helper()

	manager, err := config.NewManager(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	err = manager.AddInstance("default", &config.Instance{
		APIKey:   "test-api-key-123456789",
		VeID:     "123456",
		Endpoint: endpoint,
	}, true)
	if err != nil {
		t.Fatalf("AddInstance() error = %v", err)
	}
	return manager
}

func TestMCPReadOnlyPayloads(t *testing.T) {
	t.Run("ssh keys default shortened", func(t *testing.T) {
		payload := sshKeysPayload("default", &client.SshKeysResponse{
			SshKeysVeid:               "ssh-rsa full-vm",
			SshKeysUser:               "ssh-rsa full-user",
			SshKeysPreferred:          "ssh-rsa full-preferred",
			ShortenedSshKeysVeid:      "ssh-rsa short-vm",
			ShortenedSshKeysUser:      "ssh-rsa short-user",
			ShortenedSshKeysPreferred: "ssh-rsa short-preferred",
		}, false)

		keys := payload["keys"].(map[string][]string)
		if keys["veid"][0] != "ssh-rsa short-vm" {
			t.Fatalf("veid key = %q", keys["veid"][0])
		}
		if payload["full"].(bool) {
			t.Fatal("full = true, want false")
		}
	})

	t.Run("ssh keys full", func(t *testing.T) {
		payload := sshKeysPayload("default", &client.SshKeysResponse{
			SshKeysVeid:          "ssh-rsa full-vm",
			SshKeysUser:          "ssh-rsa full-user",
			SshKeysPreferred:     "ssh-rsa full-preferred",
			ShortenedSshKeysVeid: "ssh-rsa short-vm",
		}, true)

		keys := payload["keys"].(map[string][]string)
		if keys["veid"][0] != "ssh-rsa full-vm" {
			t.Fatalf("veid key = %q", keys["veid"][0])
		}
	})

	t.Run("os templates", func(t *testing.T) {
		payload := availableOSPayload("default", &client.AvailableOSResponse{
			Installed: "debian-12-x86_64",
			Templates: []string{
				"debian-12-x86_64",
				"ubuntu-24.04-x86_64",
			},
		})

		if payload["total_templates"] != 2 {
			t.Fatalf("total_templates = %v", payload["total_templates"])
		}
	})

	t.Run("rate limit", func(t *testing.T) {
		payload := rateLimitPayload("default", &client.RateLimitStatus{
			RemainingPoints15Min: 997,
			RemainingPoints24H:   19852,
		})

		if payload["remaining_points_15min"] != 997 {
			t.Fatalf("remaining_points_15min = %v", payload["remaining_points_15min"])
		}
	})

	t.Run("migration locations", func(t *testing.T) {
		payload := migrationLocationsPayload("default", &client.MigrateLocationsResponse{
			CurrentLocation: "usca_2",
			Locations:       []string{"usca_2", "usny_6"},
			Descriptions: map[string]string{
				"usca_2": "US: California",
				"usny_6": "US: New York",
			},
			DataTransferMultipliers: map[string]int{
				"usca_2": 1,
				"usny_6": 2,
			},
		})

		if payload["total"] != 2 {
			t.Fatalf("total = %v", payload["total"])
		}
	})

	t.Run("private ip", func(t *testing.T) {
		payload := privateIPAvailablePayload("default", &client.PrivateIPAvailableResponse{
			AvailableIPs: []string{"10.0.0.10", "10.0.0.11"},
		})

		if payload["total"] != 2 {
			t.Fatalf("total = %v", payload["total"])
		}
	})

	t.Run("suspensions", func(t *testing.T) {
		payload := suspensionDetailsPayload("default", &client.SuspensionDetailsResponse{
			SuspensionCount:  1,
			TotalAbusePoints: 20,
			MaxAbusePoints:   60,
			Suspensions: []client.SuspensionRecord{
				{RecordID: 123, Flag: "spam", AbusePoints: 20},
			},
			Evidence: map[string]string{"456": "sample evidence"},
		})

		if payload["suspension_count"] != 1 {
			t.Fatalf("suspension_count = %v", payload["suspension_count"])
		}
	})

	t.Run("policy violations", func(t *testing.T) {
		payload := policyViolationsPayload("default", &client.PolicyViolationsResponse{
			TotalAbusePoints: 10,
			MaxAbusePoints:   60,
			PolicyViolations: []client.PolicyViolationRecord{
				{RecordID: 789, Flag: "policy", AbusePoints: 10},
			},
		})

		if payload["total_abuse_points"] != 10 {
			t.Fatalf("total_abuse_points = %v", payload["total_abuse_points"])
		}
	})

	t.Run("notifications", func(t *testing.T) {
		payload := notificationPreferencesPayload("default", &client.NotificationPreferencesResponse{
			NotificationEmail: "user@example.com",
			EmailPreferences: map[string]map[string]client.NotificationPreference{
				"service": {
					"maintenance": {IsEnabled: 1},
					"billing":     {IsEnabled: 0},
				},
			},
		})

		if payload["total_categories"] != 1 {
			t.Fatalf("total_categories = %v", payload["total_categories"])
		}
		if payload["total_preference_ids"] != 2 {
			t.Fatalf("total_preference_ids = %v", payload["total_preference_ids"])
		}
	})
}

func TestCallReadOnlyTool(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := strings.TrimPrefix(r.URL.Path, "/"); got != "getRateLimitStatus" {
				t.Fatalf("endpoint = %s, want getRateLimitStatus", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"error":0,"remaining_points_15min":997,"remaining_points_24h":19852}`))
		}))
		defer server.Close()

		manager := newMCPTestManager(t, server.URL)
		result, err := callReadOnlyTool(context.Background(), manager, "", "default", "get rate limit",
			func(ctx context.Context, c *client.Client) (*client.RateLimitStatus, error) {
				return c.GetRateLimitStatus(ctx)
			},
			rateLimitPayload,
		)
		if err != nil {
			t.Fatalf("callReadOnlyTool() error = %v", err)
		}
		if result.IsError {
			t.Fatalf("result.IsError = true")
		}

		payload := result.StructuredContent.(map[string]any)
		if payload["instance"] != "default" {
			t.Fatalf("instance = %v", payload["instance"])
		}
	})

	t.Run("resolve error", func(t *testing.T) {
		manager, err := config.NewManager(filepath.Join(t.TempDir(), "config.yaml"))
		if err != nil {
			t.Fatalf("NewManager() error = %v", err)
		}

		result, err := callReadOnlyTool(context.Background(), manager, "", "default", "get rate limit",
			func(ctx context.Context, c *client.Client) (*client.RateLimitStatus, error) {
				return c.GetRateLimitStatus(ctx)
			},
			rateLimitPayload,
		)
		if err != nil {
			t.Fatalf("callReadOnlyTool() error = %v", err)
		}
		if !result.IsError {
			t.Fatalf("result.IsError = false")
		}
	})

	t.Run("api error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"error":700005,"message":"Authentication failure"}`))
		}))
		defer server.Close()

		manager := newMCPTestManager(t, server.URL)
		result, err := callReadOnlyTool(context.Background(), manager, "", "default", "get rate limit",
			func(ctx context.Context, c *client.Client) (*client.RateLimitStatus, error) {
				return c.GetRateLimitStatus(ctx)
			},
			rateLimitPayload,
		)
		if err != nil {
			t.Fatalf("callReadOnlyTool() error = %v", err)
		}
		if !result.IsError {
			t.Fatalf("result.IsError = false")
		}
	})
}

func TestCallReadOnlyToolUsesServerDefaultInstance(t *testing.T) {
	seenVEIDs := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenVEIDs = append(seenVEIDs, r.URL.Query().Get("veid"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":0,"remaining_points_15min":997,"remaining_points_24h":19852}`))
	}))
	defer server.Close()

	manager, err := config.NewManager(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	if err := manager.AddInstance("primary", &config.Instance{
		APIKey:   "primary-api-key-123456789",
		VeID:     "111111",
		Endpoint: server.URL,
	}, true); err != nil {
		t.Fatalf("AddInstance(primary) error = %v", err)
	}
	if err := manager.AddInstance("secondary", &config.Instance{
		APIKey:   "secondary-api-key-123456789",
		VeID:     "222222",
		Endpoint: server.URL,
	}, false); err != nil {
		t.Fatalf("AddInstance(secondary) error = %v", err)
	}

	result, err := callReadOnlyTool(context.Background(), manager, "", "secondary", "get rate limit",
		func(ctx context.Context, c *client.Client) (*client.RateLimitStatus, error) {
			return c.GetRateLimitStatus(ctx)
		},
		rateLimitPayload,
	)
	if err != nil {
		t.Fatalf("callReadOnlyTool() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("result.IsError = true")
	}
	payload := result.StructuredContent.(map[string]any)
	if payload["instance"] != "secondary" {
		t.Fatalf("instance = %v, want secondary", payload["instance"])
	}
	if len(seenVEIDs) != 1 || seenVEIDs[0] != "222222" {
		t.Fatalf("veids = %v, want [222222]", seenVEIDs)
	}

	result, err = callReadOnlyTool(context.Background(), manager, "primary", "secondary", "get rate limit",
		func(ctx context.Context, c *client.Client) (*client.RateLimitStatus, error) {
			return c.GetRateLimitStatus(ctx)
		},
		rateLimitPayload,
	)
	if err != nil {
		t.Fatalf("callReadOnlyTool() with explicit instance error = %v", err)
	}
	if result.IsError {
		t.Fatalf("result.IsError = true for explicit instance")
	}
	payload = result.StructuredContent.(map[string]any)
	if payload["instance"] != "primary" {
		t.Fatalf("instance = %v, want primary", payload["instance"])
	}
	if len(seenVEIDs) != 2 || seenVEIDs[1] != "111111" {
		t.Fatalf("veids = %v, want second call to use 111111", seenVEIDs)
	}
}
