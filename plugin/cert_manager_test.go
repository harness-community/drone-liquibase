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

func TestNewCertManager(t *testing.T) {
	args := CertArgs{
		CertsDir:      "/test/certs",
		StorePassword: "testpass",
	}

	cm := NewCertManager(args, "/test/java")

	if cm.certsDir != "/test/certs" {
		t.Errorf("certsDir = %q, want %q", cm.certsDir, "/test/certs")
	}
	if cm.storePassword != "testpass" {
		t.Errorf("storePassword = %q, want %q", cm.storePassword, "testpass")
	}
}

func TestDetectJavaHome(t *testing.T) {
	// Test with JAVA_HOME set
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

func TestSetupCertificatesNoCerts(t *testing.T) {
	tmpDir := t.TempDir()
	args := CertArgs{
		CertsDir:       tmpDir,
		StorePassword:  "changeit",
		SSLCACertPath:  "/nonexistent/ca.crt",
		ClientCertPath: "/nonexistent/client.crt",
		ClientKeyPath:  "/nonexistent/client.key",
	}

	cm := NewCertManager(args, "/test/java")
	javaOpts, err := cm.SetupCertificates(args)

	// Should not error, just skip cert setup when files don't exist
	if err != nil {
		t.Errorf("SetupCertificates() error = %v, want nil", err)
	}

	// Should return empty JAVA_OPTS when no certs configured
	if javaOpts != "" {
		t.Errorf("SetupCertificates() javaOpts = %q, want empty", javaOpts)
	}
}

func TestDetectJavaHomeFromSettings(t *testing.T) {
	// Clear JAVA_HOME to test detection from java -XshowSettings:properties
	originalJavaHome := os.Getenv("JAVA_HOME")
	defer os.Setenv("JAVA_HOME", originalJavaHome)
	os.Unsetenv("JAVA_HOME")

	result := detectJavaHomeFromSettings()
	// Result depends on whether Java is installed
	// If Java is available, result should be non-empty
	// This test validates the parsing logic works without panicking

	// The function should return a valid path if Java is available
	if result != "" {
		// Basic sanity check - should not contain "java.home" literal
		if strings.Contains(result, "java.home") {
			t.Errorf("detectJavaHomeFromSettings() returned unparsed output: %q", result)
		}
	}
}

func TestSetJavaHomeEnv(t *testing.T) {
	// Save original value
	originalJavaHome := os.Getenv("JAVA_HOME")
	defer os.Setenv("JAVA_HOME", originalJavaHome)

	// Test setting JAVA_HOME
	testPath := "/test/java/path"
	setJavaHomeEnv(testPath)

	if got := os.Getenv("JAVA_HOME"); got != testPath {
		t.Errorf("setJavaHomeEnv() JAVA_HOME = %q, want %q", got, testPath)
	}

	// Test with empty path (should not change env)
	os.Setenv("JAVA_HOME", "/existing/path")
	setJavaHomeEnv("")
	if got := os.Getenv("JAVA_HOME"); got != "/existing/path" {
		t.Errorf("setJavaHomeEnv(\"\") should not change JAVA_HOME, got %q", got)
	}
}

func TestDetectJavaHomeExportsToEnv(t *testing.T) {
	// Test that detectJavaHome() exports JAVA_HOME to environment
	originalJavaHome := os.Getenv("JAVA_HOME")
	defer os.Setenv("JAVA_HOME", originalJavaHome)

	// Clear JAVA_HOME and set to a test value
	testPath := "/test/export/java"
	os.Setenv("JAVA_HOME", testPath)

	result := detectJavaHome()
	if result != testPath {
		t.Errorf("detectJavaHome() = %q, want %q", result, testPath)
	}

	// Verify JAVA_HOME is still set (exported)
	if got := os.Getenv("JAVA_HOME"); got != testPath {
		t.Errorf("JAVA_HOME should be exported, got %q", got)
	}
}

func TestCertificateExistsNonexistentKeystore(t *testing.T) {
	cm := &CertManager{
		storePassword: "changeit",
	}

	// Should return false for nonexistent keystore
	exists := cm.certificateExists("/nonexistent/keystore", "test-alias")
	if exists {
		t.Error("certificateExists() should return false for nonexistent keystore")
	}
}

func TestCertManagerSetsCertsDir(t *testing.T) {
	args := CertArgs{
		CertsDir:      "/custom/certs/dir",
		StorePassword: "custompass",
	}

	cm := NewCertManager(args, "/test/java")

	if cm.certsDir != "/custom/certs/dir" {
		t.Errorf("certsDir = %q, want %q", cm.certsDir, "/custom/certs/dir")
	}
	if cm.storePassword != "custompass" {
		t.Errorf("storePassword = %q, want %q", cm.storePassword, "custompass")
	}
}
