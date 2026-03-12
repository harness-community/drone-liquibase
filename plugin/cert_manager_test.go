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
	"testing"
)

func TestNewCertManager(t *testing.T) {
	args := CertArgs{
		CertsDir:      "/test/certs",
		StorePassword: "testpass",
	}

	cm := NewCertManager(args)

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

	cm := NewCertManager(args)
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
