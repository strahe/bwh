package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

type fakeISOAPI struct {
	service   *client.ServiceInfo
	mounted   []string
	unmounted int
}

func (f *fakeISOAPI) GetServiceInfo(context.Context) (*client.ServiceInfo, error) {
	return f.service, nil
}

func (f *fakeISOAPI) MountISO(_ context.Context, iso string) error {
	f.mounted = append(f.mounted, iso)
	return nil
}

func (f *fakeISOAPI) UnmountISO(context.Context) error {
	f.unmounted++
	return nil
}

func TestRunISOSafety(t *testing.T) {
	api := &fakeISOAPI{service: &client.ServiceInfo{
		AvailableISOs: []string{"ubuntu.iso", "debian.iso"},
		ISO1:          "debian.iso",
	}}

	out := captureStdout(t, func() {
		if err := runMountISO(context.Background(), api, "test", "ubuntu.iso", true, false, confirmNo); err != nil {
			t.Fatalf("runMountISO() error = %v", err)
		}
	})
	if len(api.mounted) != 0 {
		t.Fatalf("mounted = %v, want none", api.mounted)
	}
	if !strings.Contains(out, "DRY RUN") {
		t.Fatalf("output missing DRY RUN:\n%s", out)
	}

	if err := runMountISO(context.Background(), api, "test", "ubuntu.iso", false, false, confirmNo); err != nil {
		t.Fatalf("runMountISO() error = %v", err)
	}
	if len(api.mounted) != 0 {
		t.Fatalf("mounted = %v, want none after cancel", api.mounted)
	}

	if err := runMountISO(context.Background(), api, "test", "ubuntu.iso", false, true, confirmNo); err != nil {
		t.Fatalf("runMountISO() error = %v", err)
	}
	if len(api.mounted) != 1 || api.mounted[0] != "ubuntu.iso" {
		t.Fatalf("mounted = %v, want [ubuntu.iso]", api.mounted)
	}

	out = captureStdout(t, func() {
		if err := runUnmountISO(context.Background(), api, "test", true, false, confirmNo); err != nil {
			t.Fatalf("runUnmountISO() error = %v", err)
		}
	})
	if api.unmounted != 0 {
		t.Fatalf("unmounted = %d, want 0", api.unmounted)
	}
	if !strings.Contains(out, "DRY RUN") {
		t.Fatalf("output missing DRY RUN:\n%s", out)
	}

	if err := runUnmountISO(context.Background(), api, "test", false, true, confirmNo); err != nil {
		t.Fatalf("runUnmountISO() error = %v", err)
	}
	if api.unmounted != 1 {
		t.Fatalf("unmounted = %d, want 1", api.unmounted)
	}

	if err := runMountISO(context.Background(), api, "test", "missing.iso", true, true, confirmNo); err == nil {
		t.Fatal("runMountISO() error = nil, want unavailable ISO error")
	}
	noopAPI := &fakeISOAPI{service: &client.ServiceInfo{AvailableISOs: []string{"debian.iso"}, ISO1: "debian.iso"}}
	if err := runMountISO(context.Background(), noopAPI, "test", "debian.iso", false, true, confirmNo); err != nil {
		t.Fatalf("runMountISO() noop error = %v", err)
	}
	if len(noopAPI.mounted) != 0 {
		t.Fatalf("mounted = %v, want none for noop", noopAPI.mounted)
	}
}

type fakeIPv6API struct {
	service *client.ServiceInfo
	added   int
	deleted []string
}

func (f *fakeIPv6API) GetServiceInfo(context.Context) (*client.ServiceInfo, error) {
	return f.service, nil
}

func (f *fakeIPv6API) AddIPv6(context.Context) (*client.IPv6AddResponse, error) {
	f.added++
	return &client.IPv6AddResponse{AssignedSubnet: "2001:db8:abcd::"}, nil
}

func (f *fakeIPv6API) DeleteIPv6(_ context.Context, subnet string) error {
	f.deleted = append(f.deleted, subnet)
	return nil
}

func TestRunIPv6Safety(t *testing.T) {
	api := &fakeIPv6API{service: &client.ServiceInfo{
		LocationIPv6Ready: true,
		PlanMaxIPv6s:      2,
		IPAddresses:       []string{"2001:db8:abcd::/64"},
	}}

	out := captureStdout(t, func() {
		if err := runIPv6Add(context.Background(), api, "test", true, false, confirmNo); err != nil {
			t.Fatalf("runIPv6Add() error = %v", err)
		}
	})
	if api.added != 0 {
		t.Fatalf("added = %d, want 0", api.added)
	}
	if !strings.Contains(out, "DRY RUN") {
		t.Fatalf("output missing DRY RUN:\n%s", out)
	}

	out = captureStdout(t, func() {
		if err := runIPv6Delete(context.Background(), api, "test", "2001:db8:abcd::", true, false, confirmNo); err != nil {
			t.Fatalf("runIPv6Delete() error = %v", err)
		}
	})
	if len(api.deleted) != 0 {
		t.Fatalf("deleted = %v, want none", api.deleted)
	}
	if !strings.Contains(out, "DRY RUN") {
		t.Fatalf("output missing DRY RUN:\n%s", out)
	}

	if err := runIPv6Add(context.Background(), api, "test", false, false, confirmNo); err != nil {
		t.Fatalf("runIPv6Add() error = %v", err)
	}
	if api.added != 0 {
		t.Fatalf("added = %d, want 0 after cancel", api.added)
	}

	if err := runIPv6Add(context.Background(), api, "test", false, true, confirmNo); err != nil {
		t.Fatalf("runIPv6Add() error = %v", err)
	}
	if api.added != 1 {
		t.Fatalf("added = %d, want 1", api.added)
	}
}

type fakePrivateIPAPI struct {
	service   *client.ServiceInfo
	available *client.PrivateIPAvailableResponse
	assigned  []string
	deleted   []string
}

func (f *fakePrivateIPAPI) GetServiceInfo(context.Context) (*client.ServiceInfo, error) {
	return f.service, nil
}

func (f *fakePrivateIPAPI) GetAvailablePrivateIPs(context.Context) (*client.PrivateIPAvailableResponse, error) {
	return f.available, nil
}

func (f *fakePrivateIPAPI) AssignPrivateIP(_ context.Context, ip string) (*client.PrivateIPAssignResponse, error) {
	f.assigned = append(f.assigned, ip)
	if ip == "" {
		ip = "10.0.0.10"
	}
	return &client.PrivateIPAssignResponse{AssignedIPs: []string{ip}}, nil
}

func (f *fakePrivateIPAPI) DeletePrivateIP(_ context.Context, ip string) error {
	f.deleted = append(f.deleted, ip)
	return nil
}

func TestRunPrivateIPSafety(t *testing.T) {
	api := &fakePrivateIPAPI{
		service: &client.ServiceInfo{
			PlanPrivateNetworkAvailable:     true,
			LocationPrivateNetworkAvailable: true,
			PrivateIPAddresses:              []string{"10.0.0.20"},
		},
		available: &client.PrivateIPAvailableResponse{AvailableIPs: []string{"10.0.0.10"}},
	}

	out := captureStdout(t, func() {
		if err := runPrivateIPAssign(context.Background(), api, "test", "10.0.0.10", true, false, confirmNo); err != nil {
			t.Fatalf("runPrivateIPAssign() error = %v", err)
		}
	})
	if len(api.assigned) != 0 {
		t.Fatalf("assigned = %v, want none", api.assigned)
	}
	if !strings.Contains(out, "DRY RUN") {
		t.Fatalf("output missing DRY RUN:\n%s", out)
	}

	out = captureStdout(t, func() {
		if err := runPrivateIPDelete(context.Background(), api, "test", "10.0.0.20", true, false, confirmNo); err != nil {
			t.Fatalf("runPrivateIPDelete() error = %v", err)
		}
	})
	if len(api.deleted) != 0 {
		t.Fatalf("deleted = %v, want none", api.deleted)
	}
	if !strings.Contains(out, "DRY RUN") {
		t.Fatalf("output missing DRY RUN:\n%s", out)
	}

	if err := runPrivateIPAssign(context.Background(), api, "test", "10.0.0.10", false, false, confirmNo); err != nil {
		t.Fatalf("runPrivateIPAssign() error = %v", err)
	}
	if len(api.assigned) != 0 {
		t.Fatalf("assigned = %v, want none after cancel", api.assigned)
	}

	if err := runPrivateIPAssign(context.Background(), api, "test", "10.0.0.10", false, true, confirmNo); err != nil {
		t.Fatalf("runPrivateIPAssign() error = %v", err)
	}
	if len(api.assigned) != 1 || api.assigned[0] != "10.0.0.10" {
		t.Fatalf("assigned = %v, want [10.0.0.10]", api.assigned)
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

func TestRunResetPasswordDefaultDryRunDoesNotCreateDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	api := &fakeResetPasswordAPI{}

	out := captureStdout(t, func() {
		if err := runResetPassword(context.Background(), api, "test", "", true, false, confirmNo); err != nil {
			t.Fatalf("runResetPassword() error = %v", err)
		}
	})
	if api.calls != 0 {
		t.Fatalf("calls = %d, want 0", api.calls)
	}
	if _, err := os.Stat(filepath.Join(home, ".bwh")); !os.IsNotExist(err) {
		t.Fatalf("default output directory should not be created during dry-run, stat error = %v", err)
	}
	if !strings.Contains(out, filepath.Join(home, ".bwh")) {
		t.Fatalf("dry-run output missing secure default directory:\n%s", out)
	}
}

func TestRunResetPasswordDefaultOutputUsesSecureBWHDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	api := &fakeResetPasswordAPI{}

	out := captureStdout(t, func() {
		if err := runResetPassword(context.Background(), api, "test", "", false, true, confirmNo); err != nil {
			t.Fatalf("runResetPassword() error = %v", err)
		}
	})
	if api.calls != 1 {
		t.Fatalf("calls = %d, want 1", api.calls)
	}

	outputDir := filepath.Join(home, ".bwh")
	info, err := os.Stat(outputDir)
	if err != nil {
		t.Fatalf("failed to stat default output directory: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("default output path is not a directory: %s", outputDir)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Fatalf("default output directory mode = %o, want 700", got)
	}

	matches, err := filepath.Glob(filepath.Join(outputDir, "password_*.txt"))
	if err != nil {
		t.Fatalf("failed to glob default output files: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("default output files = %v, want one password file", matches)
	}
	content, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("failed to read default output file: %v", err)
	}
	if !strings.Contains(string(content), "Password: secret") {
		t.Fatalf("default output file missing password:\n%s", content)
	}
	if !strings.Contains(out, matches[0]) {
		t.Fatalf("output missing saved file path:\n%s", out)
	}
}

func TestRunResetPasswordPreparesOutputBeforeAPI(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/password.txt"
	api := &fakeResetPasswordAPI{
		beforeCall: func() error {
			matches, err := filepath.Glob(filepath.Join(dir, ".password.txt.tmp-*"))
			if err != nil {
				return err
			}
			if len(matches) != 1 {
				return fmt.Errorf("temporary output files = %v, want one", matches)
			}
			info, err := os.Stat(matches[0])
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

func TestRunResetPasswordPreservesRecoveryFileAfterAPISuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "password.txt")
	api := &fakeResetPasswordAPI{
		beforeCall: func() error {
			return os.Mkdir(path, 0o700)
		},
	}

	var err error
	out := captureStdout(t, func() {
		err = runResetPassword(context.Background(), api, "test", path, false, true, confirmNo)
	})
	if err == nil {
		t.Fatal("runResetPassword() error = nil, want save error after API success")
	}
	if !strings.Contains(err.Error(), "root password was reset") {
		t.Fatalf("error does not make remote side effect clear: %v", err)
	}
	if api.calls != 1 {
		t.Fatalf("calls = %d, want 1", api.calls)
	}
	if !strings.Contains(out, "Root password was reset") || !strings.Contains(out, "Temporary password file preserved") {
		t.Fatalf("output missing recovery warning:\n%s", out)
	}

	matches, globErr := filepath.Glob(filepath.Join(dir, ".password.txt.tmp-*"))
	if globErr != nil {
		t.Fatalf("failed to glob temporary output files: %v", globErr)
	}
	if len(matches) != 1 {
		t.Fatalf("temporary output files = %v, want one preserved file", matches)
	}
	content, readErr := os.ReadFile(matches[0])
	if readErr != nil {
		t.Fatalf("failed to read preserved temporary output file: %v", readErr)
	}
	if !strings.Contains(string(content), "Password: secret") {
		t.Fatalf("preserved temporary output missing password:\n%s", content)
	}
}

func TestPasswordOutputPreservesExistingFileOnWriteError(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/password.txt"
	if err := os.WriteFile(path, []byte("old-password"), 0o600); err != nil {
		t.Fatalf("failed to seed output file: %v", err)
	}

	output, err := openPasswordOutputFile(path, true)
	if err != nil {
		t.Fatalf("openPasswordOutputFile() error = %v", err)
	}
	if err := output.file.Close(); err != nil {
		t.Fatalf("failed to close temp output file: %v", err)
	}

	if err := output.write("new-password"); err == nil {
		t.Fatal("passwordOutputFile.write() error = nil, want error")
	}
	output.abort()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if string(content) != "old-password" {
		t.Fatalf("output file content = %q, want old password preserved", content)
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
		if err := runReinstall(context.Background(), api, "test", "ubuntu-24.04-x86_64", false, true, false, func(string, string, string) (bool, error) {
			t.Fatal("confirm called during dry-run")
			return false, nil
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

	if err := runReinstall(context.Background(), api, "test", "ubuntu-24.04-x86_64", false, false, true, func(string, string, string) (bool, error) {
		return false, nil
	}); err != nil {
		t.Fatalf("runReinstall() error = %v", err)
	}
	if len(api.reinstalled) != 1 || api.reinstalled[0] != "ubuntu-24.04-x86_64" {
		t.Fatalf("reinstalled = %v, want [ubuntu-24.04-x86_64]", api.reinstalled)
	}
}

type fakeSnapshotAPI struct {
	snapshots []client.SnapshotInfo
	created   []string
	deleted   []string
	restored  []string
	sticky    []string
	exported  []string
	imported  []string
}

func (f *fakeSnapshotAPI) CreateSnapshot(_ context.Context, description string) (*client.CreateSnapshotResponse, error) {
	f.created = append(f.created, description)
	return &client.CreateSnapshotResponse{}, nil
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

func (f *fakeSnapshotAPI) ToggleSnapshotSticky(_ context.Context, fileName string, sticky bool) error {
	f.sticky = append(f.sticky, fmt.Sprintf("%s=%v", fileName, sticky))
	return nil
}

func (f *fakeSnapshotAPI) ExportSnapshot(_ context.Context, fileName string) (*client.SnapshotExportResponse, error) {
	f.exported = append(f.exported, fileName)
	return &client.SnapshotExportResponse{Token: "export-token"}, nil
}

func (f *fakeSnapshotAPI) ImportSnapshot(_ context.Context, sourceVeid, sourceToken string) error {
	f.imported = append(f.imported, sourceVeid+"="+sourceToken)
	return nil
}

func TestRunSnapshotCreateAndStickySafety(t *testing.T) {
	api := &fakeSnapshotAPI{snapshots: []client.SnapshotInfo{{FileName: "snap.tar.gz", OS: "debian", Sticky: false}}}

	out := captureStdout(t, func() {
		if err := runSnapshotCreate(context.Background(), api, "test", "desc", true, false, confirmNo); err != nil {
			t.Fatalf("runSnapshotCreate() error = %v", err)
		}
	})
	if len(api.created) != 0 {
		t.Fatalf("created = %v, want none", api.created)
	}
	if !strings.Contains(out, "DRY RUN") {
		t.Fatalf("output missing DRY RUN:\n%s", out)
	}

	if err := runSnapshotCreate(context.Background(), api, "test", "desc", false, false, confirmNo); err != nil {
		t.Fatalf("runSnapshotCreate() error = %v", err)
	}
	if len(api.created) != 0 {
		t.Fatalf("created = %v, want none after cancel", api.created)
	}

	out = captureStdout(t, func() {
		if err := runToggleSnapshotSticky(context.Background(), api, "test", "snap.tar.gz", true, true, false, confirmNo); err != nil {
			t.Fatalf("runToggleSnapshotSticky() error = %v", err)
		}
	})
	if len(api.sticky) != 0 {
		t.Fatalf("sticky = %v, want none", api.sticky)
	}
	if !strings.Contains(out, "DRY RUN") {
		t.Fatalf("output missing DRY RUN:\n%s", out)
	}

	if err := runToggleSnapshotSticky(context.Background(), api, "test", "snap.tar.gz", true, false, true, confirmNo); err != nil {
		t.Fatalf("runToggleSnapshotSticky() error = %v", err)
	}
	if len(api.sticky) != 1 || api.sticky[0] != "snap.tar.gz=true" {
		t.Fatalf("sticky = %v, want [snap.tar.gz=true]", api.sticky)
	}
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

func TestRunSnapshotExportImportSafety(t *testing.T) {
	token := "0123456789abcdef0123456789abcdef01234567"
	api := &fakeSnapshotAPI{snapshots: []client.SnapshotInfo{{FileName: "snap.tar.gz", OS: "debian"}}}

	out := captureStdout(t, func() {
		if err := runSnapshotExport(context.Background(), api, "test", "12345", "snap.tar.gz", true, false, confirmNo); err != nil {
			t.Fatalf("runSnapshotExport() error = %v", err)
		}
	})
	if len(api.exported) != 0 {
		t.Fatalf("exported = %v, want none", api.exported)
	}
	if !strings.Contains(out, "DRY RUN") {
		t.Fatalf("output missing DRY RUN:\n%s", out)
	}

	if err := runSnapshotExport(context.Background(), api, "test", "12345", "snap.tar.gz", false, false, confirmNo); err != nil {
		t.Fatalf("runSnapshotExport() error = %v", err)
	}
	if len(api.exported) != 0 {
		t.Fatalf("exported = %v, want none after cancel", api.exported)
	}

	if err := runSnapshotExport(context.Background(), api, "test", "12345", "snap.tar.gz", false, true, confirmNo); err != nil {
		t.Fatalf("runSnapshotExport() error = %v", err)
	}
	if len(api.exported) != 1 || api.exported[0] != "snap.tar.gz" {
		t.Fatalf("exported = %v, want [snap.tar.gz]", api.exported)
	}

	out = captureStdout(t, func() {
		if err := runSnapshotImport(context.Background(), api, "test", "12345", token, true, false, confirmNo); err != nil {
			t.Fatalf("runSnapshotImport() error = %v", err)
		}
	})
	if len(api.imported) != 0 {
		t.Fatalf("imported = %v, want none", api.imported)
	}
	if strings.Contains(out, token) {
		t.Fatalf("dry-run output leaked full source token:\n%s", out)
	}
	if !strings.Contains(out, "0123...4567") {
		t.Fatalf("dry-run output missing masked source token:\n%s", out)
	}

	if err := runSnapshotImport(context.Background(), api, "test", "12345", token, false, false, confirmNo); err != nil {
		t.Fatalf("runSnapshotImport() error = %v", err)
	}
	if len(api.imported) != 0 {
		t.Fatalf("imported = %v, want none after cancel", api.imported)
	}

	if err := runSnapshotImport(context.Background(), api, "test", "12345", token, false, true, confirmNo); err != nil {
		t.Fatalf("runSnapshotImport() error = %v", err)
	}
	if len(api.imported) != 1 || api.imported[0] != "12345="+token {
		t.Fatalf("imported = %v, want source pair", api.imported)
	}
}

type fakeMigrationAPI struct {
	locations *client.MigrateLocationsResponse
	started   []string
}

func (f *fakeMigrationAPI) GetMigrateLocations(context.Context) (*client.MigrateLocationsResponse, error) {
	return f.locations, nil
}

func (f *fakeMigrationAPI) StartMigrationWithTimeout(_ context.Context, locationID string, timeout time.Duration) (*client.MigrateStartResponse, error) {
	f.started = append(f.started, fmt.Sprintf("%s@%s", locationID, timeout))
	return &client.MigrateStartResponse{
		NotificationEmail: "ops@example.com",
		NewIPs:            []string{"192.0.2.10", "2001:db8::1"},
	}, nil
}

func TestRunMigrateStartSafety(t *testing.T) {
	api := &fakeMigrationAPI{locations: &client.MigrateLocationsResponse{
		CurrentLocation: "us-east",
		Locations:       []string{"us-west", "eu"},
		Descriptions:    map[string]string{"us-west": "US West"},
	}}

	out := captureStdout(t, func() {
		if err := runMigrateStart(context.Background(), api, "test", "us-west", 15*time.Minute, false, true, false, confirmNo); err != nil {
			t.Fatalf("runMigrateStart() error = %v", err)
		}
	})
	if len(api.started) != 0 {
		t.Fatalf("started = %v, want none", api.started)
	}
	if !strings.Contains(out, "DRY RUN") {
		t.Fatalf("output missing DRY RUN:\n%s", out)
	}

	if err := runMigrateStart(context.Background(), api, "test", "us-west", 15*time.Minute, false, false, false, confirmNo); err != nil {
		t.Fatalf("runMigrateStart() error = %v", err)
	}
	if len(api.started) != 0 {
		t.Fatalf("started = %v, want none after cancel", api.started)
	}

	if err := runMigrateStart(context.Background(), api, "test", "us-west", 15*time.Minute, false, false, true, confirmNo); err != nil {
		t.Fatalf("runMigrateStart() error = %v", err)
	}
	if len(api.started) != 1 || api.started[0] != "us-west@15m0s" {
		t.Fatalf("started = %v, want [us-west@15m0s]", api.started)
	}

	noopAPI := &fakeMigrationAPI{locations: &client.MigrateLocationsResponse{CurrentLocation: "us-west", Locations: []string{"us-west"}}}
	if err := runMigrateStart(context.Background(), noopAPI, "test", "us-west", 15*time.Minute, false, false, true, confirmNo); err != nil {
		t.Fatalf("runMigrateStart() noop error = %v", err)
	}
	if len(noopAPI.started) != 0 {
		t.Fatalf("started = %v, want none for noop", noopAPI.started)
	}

	if err := runMigrateStart(context.Background(), api, "test", "missing", 15*time.Minute, false, true, true, confirmNo); err == nil {
		t.Fatal("runMigrateStart() error = nil, want unavailable location error")
	}
}
