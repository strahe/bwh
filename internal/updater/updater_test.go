package updater

import (
	"runtime"
	"testing"
)

func TestCleanVersion(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"v1.2.3", "1.2.3"},
		{"1.2.3", "1.2.3"},
		{"v1.2.3-beta", "1.2.3"},
		{"1.2.3-4-g123abc", "1.2.3"},
		{"v2.0.0-rc1", "2.0.0"},
		{"0.1.0-dev", "0.1.0"},
		{"v1.0.0-1-g3eaeb94-dirty", "1.0.0"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := cleanVersion(tc.input)
			if result != tc.expected {
				t.Errorf("cleanVersion(%q) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestParseVersion(t *testing.T) {
	testCases := []struct {
		input         string
		expectedMajor int
		expectedMinor int
		expectedPatch int
		expectError   bool
	}{
		{"1.2.3", 1, 2, 3, false},
		{"0.1.0", 0, 1, 0, false},
		{"10.20.30", 10, 20, 30, false},
		{"1.2", 1, 2, 0, false},
		{"5", 5, 0, 0, false},
		{"1.2.3.4", 0, 0, 0, true}, // Too many parts
		{"", 0, 0, 0, true},        // Empty string
		{"a.b.c", 0, 0, 0, true},   // Non-numeric
		{"1.b.3", 0, 0, 0, true},   // Mixed numeric/non-numeric
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			major, minor, patch, err := parseVersion(tc.input)

			if tc.expectError {
				if err == nil {
					t.Errorf("parseVersion(%q) expected error but got none", tc.input)
				}
				return
			}

			if err != nil {
				t.Errorf("parseVersion(%q) unexpected error: %v", tc.input, err)
				return
			}

			if major != tc.expectedMajor || minor != tc.expectedMinor || patch != tc.expectedPatch {
				t.Errorf("parseVersion(%q) = (%d, %d, %d), expected (%d, %d, %d)",
					tc.input, major, minor, patch, tc.expectedMajor, tc.expectedMinor, tc.expectedPatch)
			}
		})
	}
}

func TestCompareSemanticVersions(t *testing.T) {
	testCases := []struct {
		versionA string
		versionB string
		expected int
	}{
		// Equal versions
		{"1.2.3", "1.2.3", 0},
		{"0.0.1", "0.0.1", 0},

		// A < B (should return -1)
		{"1.2.3", "1.2.4", -1},
		{"1.2.3", "1.3.0", -1},
		{"1.2.3", "2.0.0", -1},
		{"0.1.0", "0.2.0", -1},
		{"1.0", "1.1", -1},
		{"1", "2", -1},

		// A > B (should return 1)
		{"1.2.4", "1.2.3", 1},
		{"1.3.0", "1.2.3", 1},
		{"2.0.0", "1.2.3", 1},
		{"0.2.0", "0.1.0", 1},
		{"1.1", "1.0", 1},
		{"2", "1", 1},

		// Mixed format versions (missing components)
		{"1.2", "1.2.0", 0},
		{"1.2", "1.2.1", -1},
		{"1.2.1", "1.2", 1},
		{"1", "1.0.0", 0},
		{"2", "1.0.0", 1},

		// Invalid versions (fallback to string comparison)
		{"invalid", "1.2.3", 1},  // "invalid" > "1.2.3" lexicographically
		{"1.2.3", "invalid", -1}, // "1.2.3" < "invalid" lexicographically
		{"abc", "def", -1},       // "abc" < "def" lexicographically
	}

	for _, tc := range testCases {
		t.Run(tc.versionA+"_vs_"+tc.versionB, func(t *testing.T) {
			result := compareSemanticVersions(tc.versionA, tc.versionB)
			if result != tc.expected {
				t.Errorf("compareSemanticVersions(%q, %q) = %d, expected %d",
					tc.versionA, tc.versionB, result, tc.expected)
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	testCases := []struct {
		versionA string
		versionB string
		expected int
	}{
		// Test with 'v' prefix and git suffixes
		{"v1.2.3", "v1.2.4", -1},
		{"v1.2.3-1-g123abc", "v1.2.3", 0}, // Git suffix should be ignored
		{"v1.2.3-dirty", "v1.2.4", -1},
		{"1.2.3-4-g123abc-dirty", "1.2.3", 0},

		// Test clean comparison
		{"v2.0.0", "v1.9.9", 1},
		{"v0.1.0", "v0.1.1", -1},
	}

	for _, tc := range testCases {
		t.Run(tc.versionA+"_vs_"+tc.versionB, func(t *testing.T) {
			result := CompareVersions(tc.versionA, tc.versionB)
			if result != tc.expected {
				t.Errorf("CompareVersions(%q, %q) = %d, expected %d",
					tc.versionA, tc.versionB, result, tc.expected)
			}
		})
	}
}

func TestGetBinaryName(t *testing.T) {
	expectedBase := "bwh"
	expectedPlatform := runtime.GOOS + "-" + runtime.GOARCH

	result := getBinaryName()

	if runtime.GOOS == "windows" {
		expected := expectedBase + "-" + expectedPlatform + ".exe"
		if result != expected {
			t.Errorf("getBinaryName() on Windows = %q, expected %q", result, expected)
		}
	} else {
		expected := expectedBase + "-" + expectedPlatform
		if result != expected {
			t.Errorf("getBinaryName() on %s = %q, expected %q", runtime.GOOS, result, expected)
		}
	}

	// Test that result contains expected components
	if !contains(result, expectedBase) {
		t.Errorf("getBinaryName() result %q should contain %q", result, expectedBase)
	}
	if !contains(result, runtime.GOOS) {
		t.Errorf("getBinaryName() result %q should contain OS %q", result, runtime.GOOS)
	}
	if !contains(result, runtime.GOARCH) {
		t.Errorf("getBinaryName() result %q should contain ARCH %q", result, runtime.GOARCH)
	}
}

// helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsInMiddle(s, substr))))
}

func containsInMiddle(s, substr string) bool {
	for i := 1; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
