package version

import (
	"strings"
	"testing"
)

func TestGetVersion(t *testing.T) {
	version := GetVersion()
	if version == "" {
		t.Error("GetVersion() should not return empty string")
	}
}

func TestGetUserAgent(t *testing.T) {
	userAgent := GetUserAgent()

	if !strings.HasPrefix(userAgent, "BWH-CLI/") {
		t.Errorf("GetUserAgent() should start with 'BWH-CLI/', got: %s", userAgent)
	}

	// Should contain the version
	if !strings.Contains(userAgent, GetVersion()) {
		t.Errorf("GetUserAgent() should contain version '%s', got: %s", GetVersion(), userAgent)
	}
}

func TestGetFullVersionInfo(t *testing.T) {
	fullInfo := GetFullVersionInfo()

	// Should contain version, build time, and commit hash
	if !strings.Contains(fullInfo, GetVersion()) {
		t.Errorf("GetFullVersionInfo() should contain version '%s'", GetVersion())
	}

	if !strings.Contains(fullInfo, GetBuildTime()) {
		t.Errorf("GetFullVersionInfo() should contain build time '%s'", GetBuildTime())
	}

	if !strings.Contains(fullInfo, GetCommitHash()) {
		t.Errorf("GetFullVersionInfo() should contain commit hash '%s'", GetCommitHash())
	}
}
