// Copyright 2025 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package plugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectJavaHome(t *testing.T) {
	originalJavaHome := os.Getenv("JAVA_HOME")
	defer os.Setenv("JAVA_HOME", originalJavaHome)

	os.Setenv("JAVA_HOME", "/test/java/home")
	result := detectJavaHome()
	if result != "/test/java/home" {
		t.Errorf("detectJavaHome() = %q, want %q", result, "/test/java/home")
	}

	// Test without JAVA_HOME (falls back to detection)
	os.Unsetenv("JAVA_HOME")
	result = detectJavaHome()
	// Result depends on system - just verify it doesn't panic
	_ = result
}

func TestDetectJavaHomeFromSettings(t *testing.T) {
	originalJavaHome := os.Getenv("JAVA_HOME")
	defer os.Setenv("JAVA_HOME", originalJavaHome)
	os.Unsetenv("JAVA_HOME")

	result := detectJavaHomeFromSettings()
	if result != "" {
		if strings.Contains(result, "java.home") {
			t.Errorf("detectJavaHomeFromSettings() returned unparsed output: %q", result)
		}
	}
}

func TestSetJavaHomeEnv(t *testing.T) {
	originalJavaHome := os.Getenv("JAVA_HOME")
	defer os.Setenv("JAVA_HOME", originalJavaHome)

	testPath := "/test/java/path"
	setJavaHomeEnv(testPath)

	if got := os.Getenv("JAVA_HOME"); got != testPath {
		t.Errorf("setJavaHomeEnv() JAVA_HOME = %q, want %q", got, testPath)
	}

	// Empty path should not change env
	os.Setenv("JAVA_HOME", "/existing/path")
	setJavaHomeEnv("")
	if got := os.Getenv("JAVA_HOME"); got != "/existing/path" {
		t.Errorf("setJavaHomeEnv(\"\") should not change JAVA_HOME, got %q", got)
	}
}

func TestDetectJavaHomeExportsToEnv(t *testing.T) {
	originalJavaHome := os.Getenv("JAVA_HOME")
	defer os.Setenv("JAVA_HOME", originalJavaHome)

	testPath := "/test/export/java"
	os.Setenv("JAVA_HOME", testPath)

	result := detectJavaHome()
	if result != testPath {
		t.Errorf("detectJavaHome() = %q, want %q", result, testPath)
	}

	if got := os.Getenv("JAVA_HOME"); got != testPath {
		t.Errorf("JAVA_HOME should be exported, got %q", got)
	}
}

func TestReadCgroupMemoryLimit(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int64
		wantErr bool
	}{
		{"valid limit 1GB", "1073741824\n", 1073741824, false},
		{"valid limit 512MB", "536870912", 536870912, false},
		{"unlimited cgroup v2", "max\n", 0, true},
		{"unlimited cgroup v1", "9223372036854771712\n", 0, true},
		{"invalid content", "not-a-number\n", 0, true},
		{"file not found", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var path string
			if tt.name == "file not found" {
				path = "/nonexistent/cgroup/path"
			} else {
				path = filepath.Join(t.TempDir(), "memory.max")
				if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
					t.Fatalf("Failed to write test file: %v", err)
				}
			}

			got, err := readCgroupMemoryLimit(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("readCgroupMemoryLimit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("readCgroupMemoryLimit() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestComputeHeapFlagsFromPaths(t *testing.T) {
	tests := []struct {
		name        string
		memoryBytes string
		heapPercent int
		want        string
	}{
		{"1GB at 50%", "1073741824", 50, "-Xms512m -Xmx512m"},
		{"2GB at 50%", "2147483648", 50, "-Xms1024m -Xmx1024m"},
		{"512MB at 50%", "536870912", 50, "-Xms256m -Xmx256m"},
		{"1GB at 25%", "1073741824", 25, "-Xms256m -Xmx256m"},
		// Below minimum threshold: 100MB * 50% = 50MB < 64MB → skip
		{"below minimum threshold", "104857600", 50, ""},
		// Exactly at threshold: 128MB * 50% = 64MB → include
		{"at minimum threshold", "134217728", 50, "-Xms64m -Xmx64m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cgroupFile := filepath.Join(t.TempDir(), "memory.max")
			if err := os.WriteFile(cgroupFile, []byte(tt.memoryBytes), 0644); err != nil {
				t.Fatalf("Failed to write cgroup file: %v", err)
			}

			got := computeHeapFlagsFromPaths(tt.heapPercent, cgroupFile)
			if got != tt.want {
				t.Errorf("computeHeapFlagsFromPaths() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestComputeHeapFlagsFromPathsFallback(t *testing.T) {
	// First path is invalid, second is valid — should use fallback
	validFile := filepath.Join(t.TempDir(), "memory.limit_in_bytes")
	if err := os.WriteFile(validFile, []byte("1073741824"), 0644); err != nil {
		t.Fatalf("Failed to write cgroup file: %v", err)
	}

	got := computeHeapFlagsFromPaths(50, "/nonexistent/path", validFile)
	if got != "-Xms512m -Xmx512m" {
		t.Errorf("computeHeapFlagsFromPaths() with fallback = %q, want %q", got, "-Xms512m -Xmx512m")
	}
}

func TestComputeHeapFlagsFromPathsNoCgroup(t *testing.T) {
	got := computeHeapFlagsFromPaths(50, "/nonexistent/v2", "/nonexistent/v1")
	if got != "" {
		t.Errorf("computeHeapFlagsFromPaths() with no cgroup = %q, want empty", got)
	}
}

func TestComputeHeapFlagsNoCgroup(t *testing.T) {
	// In the test environment the real cgroup paths do not exist, so the
	// function must return an empty string rather than panic or error.
	got := computeHeapFlags(50)
	if got != "" {
		t.Errorf("computeHeapFlags() without cgroup = %q, want empty", got)
	}
}

func TestBuildJavaOpts(t *testing.T) {
	t.Run("static flags always prepended", func(t *testing.T) {
		got := buildJavaOpts(50)
		if !strings.HasPrefix(got, staticJVMFlags) {
			t.Errorf("buildJavaOpts() = %q, want prefix %q", got, staticJVMFlags)
		}
	})

	t.Run("empty extra opts are ignored", func(t *testing.T) {
		got := buildJavaOpts(50, "", "")
		if got != staticJVMFlags {
			t.Errorf("buildJavaOpts() = %q, want %q", got, staticJVMFlags)
		}
	})

	t.Run("non-empty extra opts appended in order", func(t *testing.T) {
		got := buildJavaOpts(50, "-Dfoo=1", "-Dbar=2")
		want := staticJVMFlags + " -Dfoo=1 -Dbar=2"
		if got != want {
			t.Errorf("buildJavaOpts() = %q, want %q", got, want)
		}
	})

	t.Run("skips heap flags when -Xms already present", func(t *testing.T) {
		got := buildJavaOpts(50, "-Xms512m -Xmx512m")
		if strings.Count(got, "-Xms") != 1 {
			t.Errorf("buildJavaOpts() added duplicate -Xms: %q", got)
		}
	})

	t.Run("skips heap flags when only -Xmx present", func(t *testing.T) {
		got := buildJavaOpts(50, "-Xmx1g")
		if strings.Count(got, "-Xmx") != 1 {
			t.Errorf("buildJavaOpts() added duplicate -Xmx: %q", got)
		}
	})
}
