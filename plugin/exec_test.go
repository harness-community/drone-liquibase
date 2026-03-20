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

func TestRunCommand(t *testing.T) {
	output, err := runCommand("echo", "hello")
	if err != nil {
		t.Fatalf("runCommand() error = %v", err)
	}
	if string(output) != "hello\n" {
		t.Errorf("runCommand() output = %q, want %q", string(output), "hello\n")
	}
}

func TestRunCommandFailure(t *testing.T) {
	_, err := runCommand("false") // 'false' command always exits with 1
	if err == nil {
		t.Error("runCommand() should error on command failure")
	}
}

func TestRunCommandNotFound(t *testing.T) {
	_, err := runCommand("nonexistent-command-xyz")
	if err == nil {
		t.Error("runCommand() should error when command not found")
	}
}

func TestRunCommandWithOutput(t *testing.T) {
	exitCode, output, err := runCommandWithOutput("echo", "test output")
	if err != nil {
		t.Fatalf("runCommandWithOutput() error = %v", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if string(output) != "test output\n" {
		t.Errorf("output = %q, want %q", string(output), "test output\n")
	}
}

func TestRunCommandWithOutputNonZeroExit(t *testing.T) {
	exitCode, _, err := runCommandWithOutput("sh", "-c", "exit 42")
	if err != nil {
		t.Fatalf("runCommandWithOutput() unexpected error = %v", err)
	}
	if exitCode != 42 {
		t.Errorf("exitCode = %d, want 42", exitCode)
	}
}

func TestSetupGoogleCloudAuthFileAlreadyExists(t *testing.T) {
	serviceAccountKeyFile := "/tmp/harness-google-application-credentials.json"

	// Create file first
	if err := os.WriteFile(serviceAccountKeyFile, []byte("existing"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(serviceAccountKeyFile)

	jsonKey := `{"new": "content"}`
	cleanup, err := setupGoogleCloudAuth(jsonKey)
	if err != nil {
		t.Errorf("setupGoogleCloudAuth() error = %v", err)
	}
	if cleanup != nil {
		t.Error("cleanup should be nil when file already exists")
	}

	// Verify file was NOT overwritten (matches bash behavior)
	content, err := os.ReadFile(serviceAccountKeyFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(content) != "existing" {
		t.Error("Existing file should not be overwritten")
	}
}
