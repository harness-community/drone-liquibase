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

	if err != nil {
		t.Errorf("SetupCertificates() error = %v, want nil", err)
	}

	if javaOpts != "" {
		t.Errorf("SetupCertificates() javaOpts = %q, want empty", javaOpts)
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
