package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/strahe/bwh/internal/version"
)

const (
	GitHubAPI            = "https://api.github.com/repos/strahe/bwh/releases/latest"
	DefaultUpdateTimeout = 5 * time.Minute  // 5 minutes default for downloads
	DefaultCheckTimeout  = 30 * time.Second // 30 seconds for API check only
	TempSuffix           = ".bwh-update"
	BackupSuffix         = ".bwh-backup"
)

type Release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []Asset   `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type UpdateInfo struct {
	CurrentVersion string
	LatestVersion  string
	HasUpdate      bool
	ReleaseDate    time.Time
	DownloadURL    string
	AssetName      string
	AssetSize      int64
}

// CheckForUpdates checks if a new version is available
func CheckForUpdates(ctx context.Context) (*UpdateInfo, error) {
	return CheckForUpdatesWithTimeout(ctx, DefaultCheckTimeout)
}

// CheckForUpdatesWithTimeout checks if a new version is available with custom timeout
func CheckForUpdatesWithTimeout(ctx context.Context, timeout time.Duration) (*UpdateInfo, error) {
	current := version.GetVersion()

	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequestWithContext(ctx, "GET", GitHubAPI, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", version.GetUserAgent())

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode release info: %w", err)
	}

	info := &UpdateInfo{
		CurrentVersion: current,
		LatestVersion:  release.TagName,
		ReleaseDate:    release.PublishedAt,
	}

	// Check if update is available
	// Handle development versions (with git info)
	cleanCurrent := current
	if strings.Contains(current, "-") {
		parts := strings.Split(current, "-")
		if len(parts) > 0 {
			cleanCurrent = parts[0]
		}
	}

	info.HasUpdate = cleanCurrent != release.TagName && !strings.HasSuffix(current, "-dev")

	if info.HasUpdate {
		// Find the appropriate asset for current platform
		assetName := getBinaryName()
		for _, asset := range release.Assets {
			if asset.Name == assetName {
				info.DownloadURL = asset.BrowserDownloadURL
				info.AssetName = asset.Name
				info.AssetSize = asset.Size
				break
			}
		}

		if info.DownloadURL == "" {
			return nil, fmt.Errorf("no binary found for platform %s/%s", runtime.GOOS, runtime.GOARCH)
		}
	}

	return info, nil
}

// PerformUpdate downloads and installs the update
func PerformUpdate(ctx context.Context, info *UpdateInfo) error {
	return PerformUpdateWithTimeout(ctx, info, DefaultUpdateTimeout)
}

// PerformUpdateWithTimeout downloads and installs the update with custom timeout
func PerformUpdateWithTimeout(ctx context.Context, info *UpdateInfo, timeout time.Duration) error {
	if !info.HasUpdate {
		return fmt.Errorf("no update available")
	}

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	// Download new binary
	tempPath := execPath + TempSuffix
	if err := downloadBinaryWithTimeout(ctx, info.DownloadURL, tempPath, timeout); err != nil {
		os.Remove(tempPath) //nolint:errcheck
		return fmt.Errorf("failed to download update: %w", err)
	}

	// Verify the download by checking size
	stat, err := os.Stat(tempPath)
	if err != nil {
		os.Remove(tempPath) //nolint:errcheck
		return fmt.Errorf("failed to verify download: %w", err)
	}
	if stat.Size() != info.AssetSize {
		os.Remove(tempPath) //nolint:errcheck
		return fmt.Errorf("download size mismatch: expected %d, got %d", info.AssetSize, stat.Size())
	}

	// Make new binary executable
	if err := os.Chmod(tempPath, 0o755); err != nil {
		os.Remove(tempPath) //nolint:errcheck
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	// Create backup of current binary
	backupPath := execPath + BackupSuffix
	if err := copyFile(execPath, backupPath); err != nil {
		os.Remove(tempPath) //nolint:errcheck
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Replace current binary
	if err := os.Rename(tempPath, execPath); err != nil {
		// Restore from backup on failure
		os.Rename(backupPath, execPath) //nolint:errcheck
		os.Remove(tempPath)             //nolint:errcheck
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	// Clean up backup (but don't fail if we can't)
	os.Remove(backupPath) //nolint:errcheck

	return nil
}

// getBinaryName returns the expected binary name for the current platform
func getBinaryName() string {
	base := "bwh"
	platform := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)

	if runtime.GOOS == "windows" {
		return fmt.Sprintf("%s-%s.exe", base, platform)
	}
	return fmt.Sprintf("%s-%s", base, platform)
}

// downloadBinaryWithTimeout downloads a binary from URL to destination with custom timeout
func downloadBinaryWithTimeout(ctx context.Context, url, dest string, timeout time.Duration) error {
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	file, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer file.Close() //nolint:errcheck

	_, err = io.Copy(file, resp.Body)
	return err
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close() //nolint:errcheck

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close() //nolint:errcheck

	_, err = io.Copy(destination, source)
	if err != nil {
		return err
	}

	// Copy permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

// CompareVersions compares two semantic version strings
// Returns: -1 if a < b, 0 if a == b, 1 if a > b
func CompareVersions(a, b string) int {
	// Simple string comparison for now
	// Could be enhanced with proper semver parsing
	if a == b {
		return 0
	}
	if a < b {
		return -1
	}
	return 1
}
