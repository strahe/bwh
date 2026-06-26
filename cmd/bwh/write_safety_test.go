package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/strahe/bwh/pkg/client"
)

type fakePowerAPI struct {
	status string
	calls  []string
}

func (f *fakePowerAPI) GetLiveServiceInfo(context.Context) (*client.LiveServiceInfo, error) {
	return &client.LiveServiceInfo{VeStatus: f.status}, nil
}

func (f *fakePowerAPI) Start(context.Context) error {
	f.calls = append(f.calls, "start")
	return nil
}

func (f *fakePowerAPI) Stop(context.Context) error {
	f.calls = append(f.calls, "stop")
	return nil
}

func (f *fakePowerAPI) Restart(context.Context) error {
	f.calls = append(f.calls, "restart")
	return nil
}

func (f *fakePowerAPI) Kill(context.Context) error {
	f.calls = append(f.calls, "kill")
	return nil
}

func TestRunVPSActionSafety(t *testing.T) {
	t.Run("dry run does not write", func(t *testing.T) {
		api := &fakePowerAPI{status: "Stopped"}
		out := captureStdout(t, func() {
			if err := runVPSAction(context.Background(), api, "test", "start", true, false, confirmNo); err != nil {
				t.Fatalf("runVPSAction() error = %v", err)
			}
		})
		if len(api.calls) != 0 {
			t.Fatalf("calls = %v, want none", api.calls)
		}
		if !strings.Contains(out, "DRY RUN") {
			t.Fatalf("output missing DRY RUN:\n%s", out)
		}
	})

	t.Run("clear stopped noop prevents write", func(t *testing.T) {
		api := &fakePowerAPI{status: "Stopped"}
		out := captureStdout(t, func() {
			if err := runVPSAction(context.Background(), api, "test", "stop", false, true, confirmYes); err != nil {
				t.Fatalf("runVPSAction() error = %v", err)
			}
		})
		if len(api.calls) != 0 {
			t.Fatalf("calls = %v, want none", api.calls)
		}
		if !strings.Contains(out, "already stopped") {
			t.Fatalf("output missing noop message:\n%s", out)
		}
	})

	t.Run("skip confirm writes", func(t *testing.T) {
		api := &fakePowerAPI{status: "Running"}
		if err := runVPSAction(context.Background(), api, "test", "restart", false, true, confirmNo); err != nil {
			t.Fatalf("runVPSAction() error = %v", err)
		}
		if len(api.calls) != 1 || api.calls[0] != "restart" {
			t.Fatalf("calls = %v, want [restart]", api.calls)
		}
	})
}

type fakeSettingsAPI struct {
	service  *client.ServiceInfo
	hosts    []string
	ptrCalls []string
}

func (f *fakeSettingsAPI) GetServiceInfo(context.Context) (*client.ServiceInfo, error) {
	return f.service, nil
}

func (f *fakeSettingsAPI) SetHostname(_ context.Context, hostname string) error {
	f.hosts = append(f.hosts, hostname)
	return nil
}

func (f *fakeSettingsAPI) SetPTR(_ context.Context, ip, ptr string) error {
	f.ptrCalls = append(f.ptrCalls, ip+"="+ptr)
	return nil
}

func TestRunSettingsSafety(t *testing.T) {
	t.Run("hostname noop", func(t *testing.T) {
		api := &fakeSettingsAPI{service: &client.ServiceInfo{Hostname: "same.example"}}
		if err := runSetHostname(context.Background(), api, "test", "same.example", false, true, confirmNo); err != nil {
			t.Fatalf("runSetHostname() error = %v", err)
		}
		if len(api.hosts) != 0 {
			t.Fatalf("hosts = %v, want none", api.hosts)
		}
	})

	t.Run("ptr dry run validates target and does not write", func(t *testing.T) {
		api := &fakeSettingsAPI{service: &client.ServiceInfo{
			IPAddresses:      []string{"192.0.2.10"},
			RDNSAPIAvailable: true,
			PTR:              map[string]string{"192.0.2.10": "old.example"},
		}}
		out := captureStdout(t, func() {
			if err := runSetPTR(context.Background(), api, "test", "192.0.2.10", "new.example", true, false, confirmNo); err != nil {
				t.Fatalf("runSetPTR() error = %v", err)
			}
		})
		if len(api.ptrCalls) != 0 {
			t.Fatalf("ptrCalls = %v, want none", api.ptrCalls)
		}
		if !strings.Contains(out, "DRY RUN") {
			t.Fatalf("output missing DRY RUN:\n%s", out)
		}
	})
}

type fakeSSHAPI struct {
	keys    *client.SshKeysResponse
	updates [][]string
}

func (f *fakeSSHAPI) GetSshKeys(context.Context) (*client.SshKeysResponse, error) {
	return f.keys, nil
}

func (f *fakeSSHAPI) UpdateSshKeys(_ context.Context, keys []string) error {
	f.updates = append(f.updates, keys)
	return nil
}

func TestRunUpdateSshKeysDryRunMasksKeys(t *testing.T) {
	fullKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFullSensitivePublicKeyMaterial user@example.com"
	api := &fakeSSHAPI{keys: &client.SshKeysResponse{}}
	out := captureStdout(t, func() {
		if err := runUpdateSshKeys(context.Background(), api, "test", []string{fullKey}, true, false, confirmNo); err != nil {
			t.Fatalf("runUpdateSshKeys() error = %v", err)
		}
	})
	if len(api.updates) != 0 {
		t.Fatalf("updates = %v, want none", api.updates)
	}
	if strings.Contains(out, "AAAAC3NzaC1lZDI1NTE5AAAAIFullSensitivePublicKeyMaterial") {
		t.Fatalf("output leaked full SSH key:\n%s", out)
	}
	if !strings.Contains(out, "DRY RUN") {
		t.Fatalf("output missing DRY RUN:\n%s", out)
	}
}

type fakeBackupCopyAPI struct {
	backups map[string]client.BackupInfo
	copies  []string
}

func (f *fakeBackupCopyAPI) ListBackups(context.Context) (*client.BackupListResponse, error) {
	return &client.BackupListResponse{Backups: f.backups}, nil
}

func (f *fakeBackupCopyAPI) CopyBackupToSnapshot(_ context.Context, backupToken string) error {
	f.copies = append(f.copies, backupToken)
	return nil
}

func TestRunBackupCopyToSnapshotMasksTokenInDryRun(t *testing.T) {
	token := "0123456789abcdef0123456789abcdef01234567"
	api := &fakeBackupCopyAPI{
		backups: map[string]client.BackupInfo{
			token: {OS: "debian-12", Size: 1024, MD5: "abc", Timestamp: 0},
		},
	}

	out := captureStdout(t, func() {
		if err := runBackupCopyToSnapshot(context.Background(), api, "test", token, true, false, confirmNo); err != nil {
			t.Fatalf("runBackupCopyToSnapshot() error = %v", err)
		}
	})
	if len(api.copies) != 0 {
		t.Fatalf("copies = %v, want none", api.copies)
	}
	if strings.Contains(out, token) {
		t.Fatalf("output leaked full backup token:\n%s", out)
	}
	if !strings.Contains(out, "0123...4567") {
		t.Fatalf("output missing masked backup token:\n%s", out)
	}
}

func TestRunBackupCopyToSnapshotMasksMissingTokenError(t *testing.T) {
	token := "0123456789abcdef0123456789abcdef01234567"
	api := &fakeBackupCopyAPI{backups: map[string]client.BackupInfo{}}

	err := runBackupCopyToSnapshot(context.Background(), api, "test", token, true, false, confirmNo)
	if err == nil {
		t.Fatal("runBackupCopyToSnapshot() error = nil, want error")
	}
	if strings.Contains(err.Error(), token) {
		t.Fatalf("error leaked full backup token: %v", err)
	}
	if !strings.Contains(err.Error(), "0123...4567") {
		t.Fatalf("error missing masked backup token: %v", err)
	}
}

type fakeResetPasswordAPI struct {
	calls      int
	beforeCall func() error
	err        error
}

func (f *fakeResetPasswordAPI) ResetRootPassword(context.Context) (*client.ResetRootPasswordResponse, error) {
	f.calls++
	if f.beforeCall != nil {
		if err := f.beforeCall(); err != nil {
			return nil, err
		}
	}
	if f.err != nil {
		return nil, f.err
	}
	return &client.ResetRootPasswordResponse{Password: "secret"}, nil
}

func TestRunResetPasswordDryRunDoesNotPromptOrWrite(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/password.txt"
	if err := os.WriteFile(path, []byte("existing"), 0o600); err != nil {
		t.Fatalf("failed to seed output file: %v", err)
	}
	api := &fakeResetPasswordAPI{}
	confirmCalled := false
	out := captureStdout(t, func() {
		err := runResetPassword(context.Background(), api, "test", path, true, false, func(string) (bool, error) {
			confirmCalled = true
			return false, nil
		})
		if err != nil {
			t.Fatalf("runResetPassword() error = %v", err)
		}
	})
	if confirmCalled {
		t.Fatal("confirm called during dry-run")
	}
	if api.calls != 0 {
		t.Fatalf("calls = %d, want 0", api.calls)
	}
	if !strings.Contains(out, "would overwrite existing file") {
		t.Fatalf("output missing overwrite preview:\n%s", out)
	}
}

func TestRunResetPasswordPreparesOutputBeforeAPI(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/password.txt"
	api := &fakeResetPasswordAPI{
		beforeCall: func() error {
			info, err := os.Stat(path)
			if err != nil {
				return err
			}
			if got := info.Mode().Perm(); got != 0o600 {
				return fmt.Errorf("mode = %o, want 600", got)
			}
			return nil
		},
	}

	if err := runResetPassword(context.Background(), api, "test", path, false, true, confirmNo); err != nil {
		t.Fatalf("runResetPassword() error = %v", err)
	}
	if api.calls != 1 {
		t.Fatalf("calls = %d, want 1", api.calls)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if !strings.Contains(string(content), "Password: secret") {
		t.Fatalf("output file missing password:\n%s", content)
	}
}

func TestRunResetPasswordRemovesNewOutputOnAPIError(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/password.txt"
	api := &fakeResetPasswordAPI{err: errors.New("boom")}

	err := runResetPassword(context.Background(), api, "test", path, false, true, confirmNo)
	if err == nil {
		t.Fatal("runResetPassword() error = nil, want error")
	}
	if api.calls != 1 {
		t.Fatalf("calls = %d, want 1", api.calls)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("output file should be removed after API error, stat error = %v", statErr)
	}
}

func TestSameStringSlicesIgnoresOrderAndPreservesCount(t *testing.T) {
	if !sameStringSlices([]string{" key-b ", "key-a"}, []string{"key-a", "key-b"}) {
		t.Fatal("sameStringSlices() should ignore order and surrounding whitespace")
	}
	if sameStringSlices([]string{"key-a", "key-a"}, []string{"key-a"}) {
		t.Fatal("sameStringSlices() should preserve duplicate counts")
	}
}

type fakeReinstallAPI struct {
	info        *client.AvailableOSResponse
	reinstalled []string
}

func (f *fakeReinstallAPI) GetAvailableOS(context.Context) (*client.AvailableOSResponse, error) {
	return f.info, nil
}

func (f *fakeReinstallAPI) ReinstallOS(_ context.Context, osTemplate string) error {
	f.reinstalled = append(f.reinstalled, osTemplate)
	return nil
}

func TestRunReinstallSafety(t *testing.T) {
	api := &fakeReinstallAPI{info: &client.AvailableOSResponse{
		Installed: "debian-12-x86_64",
		Templates: []string{
			"debian-12-x86_64",
			"ubuntu-24.04-x86_64",
		},
	}}
	out := captureStdout(t, func() {
		if err := runReinstall(context.Background(), api, "test", "ubuntu-24.04-x86_64", false, true, false, func(string, string, string) bool {
			t.Fatal("confirm called during dry-run")
			return false
		}); err != nil {
			t.Fatalf("runReinstall() error = %v", err)
		}
	})
	if len(api.reinstalled) != 0 {
		t.Fatalf("reinstalled = %v, want none", api.reinstalled)
	}
	if !strings.Contains(out, "DRY RUN") {
		t.Fatalf("output missing DRY RUN:\n%s", out)
	}

	if err := runReinstall(context.Background(), api, "test", "ubuntu-24.04-x86_64", false, false, true, func(string, string, string) bool {
		return false
	}); err != nil {
		t.Fatalf("runReinstall() error = %v", err)
	}
	if len(api.reinstalled) != 1 || api.reinstalled[0] != "ubuntu-24.04-x86_64" {
		t.Fatalf("reinstalled = %v, want [ubuntu-24.04-x86_64]", api.reinstalled)
	}
}

type fakeSnapshotAPI struct {
	snapshots []client.SnapshotInfo
	deleted   []string
	restored  []string
}

func (f *fakeSnapshotAPI) ListSnapshots(context.Context) (*client.SnapshotListResponse, error) {
	return &client.SnapshotListResponse{Snapshots: f.snapshots}, nil
}

func (f *fakeSnapshotAPI) DeleteSnapshot(_ context.Context, fileName string) error {
	f.deleted = append(f.deleted, fileName)
	return nil
}

func (f *fakeSnapshotAPI) RestoreSnapshot(_ context.Context, fileName string) error {
	f.restored = append(f.restored, fileName)
	return nil
}

func TestRunSnapshotDeleteAndRestoreSafety(t *testing.T) {
	api := &fakeSnapshotAPI{snapshots: []client.SnapshotInfo{{FileName: "snap.tar.gz", OS: "debian"}}}
	out := captureStdout(t, func() {
		if err := runSnapshotDelete(context.Background(), api, "test", "snap.tar.gz", true, false, confirmNo); err != nil {
			t.Fatalf("runSnapshotDelete() error = %v", err)
		}
	})
	if len(api.deleted) != 0 {
		t.Fatalf("deleted = %v, want none", api.deleted)
	}
	if !strings.Contains(out, "DRY RUN") {
		t.Fatalf("output missing DRY RUN:\n%s", out)
	}

	if err := runSnapshotRestore(context.Background(), api, "test", "snap.tar.gz", false, false, confirmNo); err != nil {
		t.Fatalf("runSnapshotRestore() error = %v", err)
	}
	if len(api.restored) != 0 {
		t.Fatalf("restored = %v, want none", api.restored)
	}
}
