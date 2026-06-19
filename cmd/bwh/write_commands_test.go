package main

import (
	"context"
	"strings"
	"testing"

	"github.com/strahe/bwh/pkg/client"
)

type fakeAbuseAPI struct {
	suspensions *client.SuspensionDetailsResponse
	policy      *client.PolicyViolationsResponse
	unsuspended []int
	resolved    []int
}

func (f *fakeAbuseAPI) GetSuspensionDetails(context.Context) (*client.SuspensionDetailsResponse, error) {
	return f.suspensions, nil
}

func (f *fakeAbuseAPI) GetPolicyViolations(context.Context) (*client.PolicyViolationsResponse, error) {
	return f.policy, nil
}

func (f *fakeAbuseAPI) Unsuspend(_ context.Context, recordID int) error {
	f.unsuspended = append(f.unsuspended, recordID)
	return nil
}

func (f *fakeAbuseAPI) ResolvePolicyViolation(_ context.Context, recordID int) error {
	f.resolved = append(f.resolved, recordID)
	return nil
}

type fakeNotificationAPI struct {
	preferences *client.NotificationPreferencesResponse
	updates     []map[string]bool
}

func (f *fakeNotificationAPI) GetNotificationPreferences(context.Context) (*client.NotificationPreferencesResponse, error) {
	return f.preferences, nil
}

func (f *fakeNotificationAPI) SetNotificationPreferences(_ context.Context, preferences map[string]bool) (*client.SetNotificationPreferencesResponse, error) {
	f.updates = append(f.updates, preferences)
	return &client.SetNotificationPreferencesResponse{
		UpdatedEmailPreferences: client.NotificationPreferenceStateMap{"security-successful-login": 1},
	}, nil
}

func TestParseRecordID(t *testing.T) {
	got, err := parseRecordID("123")
	if err != nil {
		t.Fatalf("parseRecordID() error = %v", err)
	}
	if got != 123 {
		t.Fatalf("parseRecordID() = %d, want 123", got)
	}

	for _, input := range []string{"", "abc", "0", "-1"} {
		t.Run(input, func(t *testing.T) {
			if _, err := parseRecordID(input); err == nil {
				t.Fatal("parseRecordID() error = nil, want error")
			}
		})
	}
}

func TestRunAbuseUnsuspendGuardsWrite(t *testing.T) {
	t.Run("dry run does not write", func(t *testing.T) {
		api := &fakeAbuseAPI{suspensions: &client.SuspensionDetailsResponse{
			Suspensions: []client.SuspensionRecord{{RecordID: 123, Flag: "spam", IsSoft: 1}},
		}}
		out := captureStdout(t, func() {
			err := runAbuseUnsuspend(context.Background(), api, "test", 123, true, false, confirmYes)
			if err != nil {
				t.Fatalf("runAbuseUnsuspend() error = %v", err)
			}
		})

		if len(api.unsuspended) != 0 {
			t.Fatalf("unsuspended = %v, want no calls", api.unsuspended)
		}
		if !strings.Contains(out, "DRY RUN") {
			t.Fatalf("output missing DRY RUN:\n%s", out)
		}
	})

	t.Run("soft gate prevents write", func(t *testing.T) {
		api := &fakeAbuseAPI{suspensions: &client.SuspensionDetailsResponse{
			Suspensions: []client.SuspensionRecord{{RecordID: 123, Flag: "spam", IsSoft: 0}},
		}}
		var err error
		captureStdout(t, func() {
			err = runAbuseUnsuspend(context.Background(), api, "test", 123, false, true, confirmYes)
		})
		if err == nil {
			t.Fatal("runAbuseUnsuspend() error = nil, want soft gate error")
		}
		if len(api.unsuspended) != 0 {
			t.Fatalf("unsuspended = %v, want no calls", api.unsuspended)
		}
	})

	t.Run("confirmation cancel prevents write", func(t *testing.T) {
		api := &fakeAbuseAPI{suspensions: &client.SuspensionDetailsResponse{
			Suspensions: []client.SuspensionRecord{{RecordID: 123, Flag: "spam", IsSoft: 1}},
		}}
		var err error
		captureStdout(t, func() {
			err = runAbuseUnsuspend(context.Background(), api, "test", 123, false, false, confirmNo)
		})
		if err != nil {
			t.Fatalf("runAbuseUnsuspend() error = %v", err)
		}
		if len(api.unsuspended) != 0 {
			t.Fatalf("unsuspended = %v, want no calls", api.unsuspended)
		}
	})

	t.Run("confirmed write", func(t *testing.T) {
		api := &fakeAbuseAPI{suspensions: &client.SuspensionDetailsResponse{
			Suspensions: []client.SuspensionRecord{{RecordID: 123, Flag: "spam", IsSoft: 1}},
		}}
		var err error
		captureStdout(t, func() {
			err = runAbuseUnsuspend(context.Background(), api, "test", 123, false, true, confirmNo)
		})
		if err != nil {
			t.Fatalf("runAbuseUnsuspend() error = %v", err)
		}
		if len(api.unsuspended) != 1 || api.unsuspended[0] != 123 {
			t.Fatalf("unsuspended = %v, want [123]", api.unsuspended)
		}
	})
}

func TestRunAbuseResolvePolicyGuardsWrite(t *testing.T) {
	t.Run("dry run does not write", func(t *testing.T) {
		api := &fakeAbuseAPI{policy: &client.PolicyViolationsResponse{
			PolicyViolations: []client.PolicyViolationRecord{{RecordID: 789, Flag: "policy", IsSoft: 1}},
		}}
		out := captureStdout(t, func() {
			err := runAbuseResolvePolicy(context.Background(), api, "test", 789, true, false, confirmYes)
			if err != nil {
				t.Fatalf("runAbuseResolvePolicy() error = %v", err)
			}
		})

		if len(api.resolved) != 0 {
			t.Fatalf("resolved = %v, want no calls", api.resolved)
		}
		if !strings.Contains(out, "DRY RUN") {
			t.Fatalf("output missing DRY RUN:\n%s", out)
		}
	})

	t.Run("soft gate prevents write", func(t *testing.T) {
		api := &fakeAbuseAPI{policy: &client.PolicyViolationsResponse{
			PolicyViolations: []client.PolicyViolationRecord{{RecordID: 789, Flag: "policy", IsSoft: 0}},
		}}
		var err error
		captureStdout(t, func() {
			err = runAbuseResolvePolicy(context.Background(), api, "test", 789, false, true, confirmYes)
		})
		if err == nil {
			t.Fatal("runAbuseResolvePolicy() error = nil, want soft gate error")
		}
		if len(api.resolved) != 0 {
			t.Fatalf("resolved = %v, want no calls", api.resolved)
		}
	})

	t.Run("confirmation cancel prevents write", func(t *testing.T) {
		api := &fakeAbuseAPI{policy: &client.PolicyViolationsResponse{
			PolicyViolations: []client.PolicyViolationRecord{{RecordID: 789, Flag: "policy", IsSoft: 1}},
		}}
		var err error
		captureStdout(t, func() {
			err = runAbuseResolvePolicy(context.Background(), api, "test", 789, false, false, confirmNo)
		})
		if err != nil {
			t.Fatalf("runAbuseResolvePolicy() error = %v", err)
		}
		if len(api.resolved) != 0 {
			t.Fatalf("resolved = %v, want no calls", api.resolved)
		}
	})

	t.Run("confirmed write", func(t *testing.T) {
		api := &fakeAbuseAPI{policy: &client.PolicyViolationsResponse{
			PolicyViolations: []client.PolicyViolationRecord{{RecordID: 789, Flag: "policy", IsSoft: 1}},
		}}
		var err error
		captureStdout(t, func() {
			err = runAbuseResolvePolicy(context.Background(), api, "test", 789, false, true, confirmNo)
		})
		if err != nil {
			t.Fatalf("runAbuseResolvePolicy() error = %v", err)
		}
		if len(api.resolved) != 1 || api.resolved[0] != 789 {
			t.Fatalf("resolved = %v, want [789]", api.resolved)
		}
	})

	t.Run("missing record", func(t *testing.T) {
		api := &fakeAbuseAPI{policy: &client.PolicyViolationsResponse{}}
		var err error
		captureStdout(t, func() {
			err = runAbuseResolvePolicy(context.Background(), api, "test", 789, false, true, confirmNo)
		})
		if err == nil {
			t.Fatal("runAbuseResolvePolicy() error = nil, want missing record error")
		}
		if len(api.resolved) != 0 {
			t.Fatalf("resolved = %v, want no calls", api.resolved)
		}
	})
}

func TestParseNotificationState(t *testing.T) {
	trueValues := []string{"on", "true", "1", "enable", "enabled", "ON"}
	for _, input := range trueValues {
		got, err := parseNotificationState(input)
		if err != nil {
			t.Fatalf("parseNotificationState(%q) error = %v", input, err)
		}
		if !got {
			t.Fatalf("parseNotificationState(%q) = false, want true", input)
		}
	}

	falseValues := []string{"off", "false", "0", "disable", "disabled", "OFF"}
	for _, input := range falseValues {
		got, err := parseNotificationState(input)
		if err != nil {
			t.Fatalf("parseNotificationState(%q) error = %v", input, err)
		}
		if got {
			t.Fatalf("parseNotificationState(%q) = true, want false", input)
		}
	}

	if _, err := parseNotificationState("maybe"); err == nil {
		t.Fatal("parseNotificationState() error = nil, want error")
	}
}

func TestRunNotificationSetGuardsWrite(t *testing.T) {
	basePrefs := &client.NotificationPreferencesResponse{
		EmailPreferences: map[string]map[string]client.NotificationPreference{
			"Security Notifications": {
				"security-successful-login": {
					FriendlyDescription: "Successful login to KiwiVM",
					IsEnabled:           0,
				},
			},
		},
	}

	t.Run("dry run does not write", func(t *testing.T) {
		api := &fakeNotificationAPI{preferences: basePrefs}
		out := captureStdout(t, func() {
			err := runNotificationSet(context.Background(), api, "test", "security-successful-login", true, true, false, confirmYes)
			if err != nil {
				t.Fatalf("runNotificationSet() error = %v", err)
			}
		})
		if len(api.updates) != 0 {
			t.Fatalf("updates = %v, want no calls", api.updates)
		}
		if !strings.Contains(out, "DRY RUN") {
			t.Fatalf("output missing DRY RUN:\n%s", out)
		}
	})

	t.Run("same state skips write", func(t *testing.T) {
		api := &fakeNotificationAPI{preferences: basePrefs}
		var err error
		captureStdout(t, func() {
			err = runNotificationSet(context.Background(), api, "test", "security-successful-login", false, false, true, confirmYes)
		})
		if err != nil {
			t.Fatalf("runNotificationSet() error = %v", err)
		}
		if len(api.updates) != 0 {
			t.Fatalf("updates = %v, want no calls", api.updates)
		}
	})

	t.Run("confirmation cancel prevents write", func(t *testing.T) {
		api := &fakeNotificationAPI{preferences: basePrefs}
		var err error
		captureStdout(t, func() {
			err = runNotificationSet(context.Background(), api, "test", "security-successful-login", true, false, false, confirmNo)
		})
		if err != nil {
			t.Fatalf("runNotificationSet() error = %v", err)
		}
		if len(api.updates) != 0 {
			t.Fatalf("updates = %v, want no calls", api.updates)
		}
	})

	t.Run("confirmed write", func(t *testing.T) {
		api := &fakeNotificationAPI{preferences: basePrefs}
		var err error
		captureStdout(t, func() {
			err = runNotificationSet(context.Background(), api, "test", "security-successful-login", true, false, true, confirmNo)
		})
		if err != nil {
			t.Fatalf("runNotificationSet() error = %v", err)
		}
		if len(api.updates) != 1 || !api.updates[0]["security-successful-login"] {
			t.Fatalf("updates = %v, want enabled update", api.updates)
		}
	})

	t.Run("unknown preference", func(t *testing.T) {
		api := &fakeNotificationAPI{preferences: basePrefs}
		var err error
		captureStdout(t, func() {
			err = runNotificationSet(context.Background(), api, "test", "missing", true, false, true, confirmNo)
		})
		if err == nil {
			t.Fatal("runNotificationSet() error = nil, want missing preference error")
		}
		if len(api.updates) != 0 {
			t.Fatalf("updates = %v, want no calls", api.updates)
		}
	})
}

func confirmYes(string) (bool, error) {
	return true, nil
}

func confirmNo(string) (bool, error) {
	return false, nil
}
