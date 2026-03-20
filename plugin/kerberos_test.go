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
	"testing"
)

func TestNewKerberosManager(t *testing.T) {
	manager := NewKerberosManager()
	if manager == nil {
		t.Fatal("NewKerberosManager() returned nil")
	}
}

func TestKerberosAuthenticateNoConfig(t *testing.T) {
	manager := NewKerberosManager()
	args := KerberosArgs{
		UserPrincipal: "", // Not configured
	}

	javaOpts, cleanup, err := manager.Authenticate(args)
	if err != nil {
		t.Errorf("Authenticate() with no config should not error, got: %v", err)
	}
	if javaOpts != "" {
		t.Errorf("javaOpts should be empty when Kerberos not configured, got: %q", javaOpts)
	}
	if cleanup != nil {
		t.Error("cleanup should be nil when Kerberos not configured")
	}
}

func TestKerberosAuthenticateMissingAuthMethod(t *testing.T) {
	manager := NewKerberosManager()
	args := KerberosArgs{
		UserPrincipal:  "user@REALM",
		Password:       "", // No password
		KeytabFilePath: "", // No keytab
	}

	_, _, err := manager.Authenticate(args)
	if err == nil {
		t.Error("Authenticate() should error when no auth method provided")
	}

	expectedMsg := "PLUGIN_KERBEROS_USER_PRINCIPAL requires an authentication method. Set either PLUGIN_KERBEROS_PASSWORD or PLUGIN_KERBEROS_KEYTAB_FILE_PATH"
	if err.Error() != expectedMsg {
		t.Errorf("Error message = %q, want %q", err.Error(), expectedMsg)
	}
}

func TestKerberosAuthenticateKeytabNotFound(t *testing.T) {
	manager := NewKerberosManager()
	args := KerberosArgs{
		UserPrincipal:  "user@REALM",
		KeytabFilePath: "/nonexistent/keytab.keytab",
	}

	_, _, err := manager.Authenticate(args)
	if err == nil {
		t.Error("Authenticate() should error when keytab file not found")
	}
}

func TestKerberosAuthenticateWithKeytabExists(t *testing.T) {
	// Create a temporary keytab file (just for existence check)
	tmpDir := t.TempDir()
	keytabPath := filepath.Join(tmpDir, "test.keytab")
	if err := os.WriteFile(keytabPath, []byte("dummy"), 0600); err != nil {
		t.Fatalf("Failed to create test keytab: %v", err)
	}

	manager := NewKerberosManager()

	// Note: This will fail kinit since we don't have a real Kerberos setup
	// but it tests the file existence check works
	args := KerberosArgs{
		UserPrincipal:  "user@REALM",
		KeytabFilePath: keytabPath,
	}

	// We expect an error because kinit will fail, but not "file not found"
	_, _, err := manager.Authenticate(args)
	if err == nil {
		t.Skip("kinit unexpectedly succeeded - real Kerberos environment?")
	}

	// The error should be about kinit failing, not file not found
	if err.Error() == "keytab file not found: "+keytabPath {
		t.Error("Should not get file not found error for existing keytab")
	}
}
