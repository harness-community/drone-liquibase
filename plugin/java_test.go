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
