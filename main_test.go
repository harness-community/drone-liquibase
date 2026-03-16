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

package main

import (
	"os/exec"
	"testing"
)

func TestValidateDependencies(t *testing.T) {
	// Save original required commands
	originalRequired := requiredCommands
	originalOptional := optionalCommands
	defer func() {
		requiredCommands = originalRequired
		optionalCommands = originalOptional
	}()

	// Test with commands that exist on most systems
	requiredCommands = []string{"echo", "ls"}
	optionalCommands = []string{}

	err := validateDependencies()
	if err != nil {
		t.Errorf("validateDependencies() with valid commands error = %v", err)
	}
}

func TestValidateDependenciesMissingRequired(t *testing.T) {
	// Save original required commands
	originalRequired := requiredCommands
	originalOptional := optionalCommands
	defer func() {
		requiredCommands = originalRequired
		optionalCommands = originalOptional
	}()

	// Test with a command that doesn't exist
	requiredCommands = []string{"nonexistent-command-xyz-12345"}
	optionalCommands = []string{}

	err := validateDependencies()
	if err == nil {
		t.Error("validateDependencies() should error for missing required command")
	}
}

func TestValidateDependenciesOptionalMissing(t *testing.T) {
	// Save original required commands
	originalRequired := requiredCommands
	originalOptional := optionalCommands
	defer func() {
		requiredCommands = originalRequired
		optionalCommands = originalOptional
	}()

	// Test with valid required and missing optional
	requiredCommands = []string{"echo"}
	optionalCommands = []string{"nonexistent-optional-xyz"}

	err := validateDependencies()
	// Should NOT error for missing optional commands
	if err != nil {
		t.Errorf("validateDependencies() should not error for missing optional command, got: %v", err)
	}
}

func TestRequiredCommandsContainsZstd(t *testing.T) {
	// zstd is required for decompressing substitution properties
	found := false
	for _, cmd := range requiredCommands {
		if cmd == "zstd" {
			found = true
			break
		}
	}

	if !found {
		t.Error("requiredCommands should include 'zstd'")
	}
}

func TestRequiredCommandsContainsKeytool(t *testing.T) {
	// keytool is required for certificate management
	found := false
	for _, cmd := range requiredCommands {
		if cmd == "keytool" {
			found = true
			break
		}
	}

	if !found {
		t.Error("requiredCommands should include 'keytool'")
	}
}

func TestRequiredCommandsContainsOpenssl(t *testing.T) {
	// openssl is required for PKCS12 generation
	found := false
	for _, cmd := range requiredCommands {
		if cmd == "openssl" {
			found = true
			break
		}
	}

	if !found {
		t.Error("requiredCommands should include 'openssl'")
	}
}

func TestOptionalCommandsContainsKinit(t *testing.T) {
	// kinit is optional (only for Kerberos)
	found := false
	for _, cmd := range optionalCommands {
		if cmd == "kinit" {
			found = true
			break
		}
	}

	if !found {
		t.Error("optionalCommands should include 'kinit'")
	}
}

func TestKinitNotRequired(t *testing.T) {
	// kinit should NOT be in required commands (only optional)
	for _, cmd := range requiredCommands {
		if cmd == "kinit" {
			t.Error("'kinit' should not be in requiredCommands, it should be optional")
		}
	}
}

func TestLookPathWorks(t *testing.T) {
	// Verify exec.LookPath works correctly (used by validateDependencies)
	_, err := exec.LookPath("echo")
	if err != nil {
		t.Skipf("echo not in PATH: %v", err)
	}

	_, err = exec.LookPath("nonexistent-command-xyz-99999")
	if err == nil {
		t.Error("LookPath should error for nonexistent command")
	}
}
