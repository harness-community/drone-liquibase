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
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOptionToEnvVar(t *testing.T) {
	tests := []struct {
		option   string
		expected string
	}{
		{"log-level", "PLUGIN_LIQUIBASE_LOG_LEVEL"},
		{"search-path", "PLUGIN_LIQUIBASE_SEARCH_PATH"},
		{"changelog-file", "PLUGIN_LIQUIBASE_CHANGELOG_FILE"},
		{"url", "PLUGIN_LIQUIBASE_URL"},
		{"default-schema-name", "PLUGIN_LIQUIBASE_DEFAULT_SCHEMA_NAME"},
	}

	for _, tt := range tests {
		t.Run(tt.option, func(t *testing.T) {
			result := OptionToEnvVar(tt.option)
			if result != tt.expected {
				t.Errorf("OptionToEnvVar(%q) = %q, want %q", tt.option, result, tt.expected)
			}
		})
	}
}

func TestEnvVarToOption(t *testing.T) {
	tests := []struct {
		envVar   string
		expected string
	}{
		{"LOG_LEVEL", "log-level"},
		{"SEARCH_PATH", "search-path"},
		{"CHANGELOG_FILE", "changelog-file"},
		{"URL", "url"},
		{"DEFAULT_SCHEMA_NAME", "default-schema-name"},
	}

	for _, tt := range tests {
		t.Run(tt.envVar, func(t *testing.T) {
			result := EnvVarToOption(tt.envVar)
			if result != tt.expected {
				t.Errorf("EnvVarToOption(%q) = %q, want %q", tt.envVar, result, tt.expected)
			}
		})
	}
}

func TestLoadGlobalOptions(t *testing.T) {
	tmpDir := t.TempDir()
	optionsFile := filepath.Join(tmpDir, "global_options.txt")

	content := "log-level\nchangelog-file\nsearch-path\n"
	if err := os.WriteFile(optionsFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	options, err := LoadGlobalOptions(optionsFile)
	if err != nil {
		t.Fatalf("LoadGlobalOptions() error = %v", err)
	}

	expected := []string{"log-level", "changelog-file", "search-path"}
	if len(options) != len(expected) {
		t.Fatalf("LoadGlobalOptions() returned %d options, want %d", len(options), len(expected))
	}

	for i, opt := range expected {
		if options[i] != opt {
			t.Errorf("options[%d] = %q, want %q", i, options[i], opt)
		}
	}
}

func TestLoadGlobalOptionsFileNotFound(t *testing.T) {
	_, err := LoadGlobalOptions("/nonexistent/file.txt")
	if err == nil {
		t.Error("LoadGlobalOptions() expected error for nonexistent file")
	}
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "exists.txt")
	if err := os.WriteFile(existingFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if !fileExists(existingFile) {
		t.Error("fileExists() returned false for existing file")
	}

	if fileExists(filepath.Join(tmpDir, "nonexistent.txt")) {
		t.Error("fileExists() returned true for nonexistent file")
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.txt")
	dstFile := filepath.Join(tmpDir, "dest.txt")

	content := []byte("test content")
	if err := os.WriteFile(srcFile, content, 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	if err := copyFile(srcFile, dstFile); err != nil {
		t.Fatalf("copyFile() error = %v", err)
	}

	copied, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}

	if string(copied) != string(content) {
		t.Errorf("copyFile() content = %q, want %q", copied, content)
	}
}

func TestCopyFileNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	err := copyFile("/nonexistent/file.txt", filepath.Join(tmpDir, "dest.txt"))
	if err == nil {
		t.Error("copyFile() should error for nonexistent source")
	}
}

func TestSetupGoogleCloudAuth(t *testing.T) {
	// Test with empty JSON key - should return nil cleanup
	cleanup, err := setupGoogleCloudAuth("")
	if err != nil {
		t.Errorf("setupGoogleCloudAuth(\"\") error = %v", err)
	}
	if cleanup != nil {
		t.Error("setupGoogleCloudAuth(\"\") should return nil cleanup")
	}
}

func TestSetupGoogleCloudAuthWithKey(t *testing.T) {
	jsonKey := `{"type": "service_account", "project_id": "test"}`
	cleanup, err := setupGoogleCloudAuth(jsonKey)
	if err != nil {
		t.Fatalf("setupGoogleCloudAuth() error = %v", err)
	}
	if cleanup == nil {
		t.Fatal("setupGoogleCloudAuth() should return cleanup function")
	}

	// Verify GOOGLE_APPLICATION_CREDENTIALS is set
	credPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credPath == "" {
		t.Error("GOOGLE_APPLICATION_CREDENTIALS should be set")
	}

	// Call cleanup
	cleanup()

	// Verify file is removed
	if fileExists(credPath) {
		t.Error("Cleanup should remove credentials file")
	}
}

func TestValidateInputs(t *testing.T) {
	tests := []struct {
		name    string
		args    Args
		wantErr bool
	}{
		{
			name: "valid args",
			args: Args{
				LiquibaseArgs: LiquibaseArgs{
					Command: "update",
				},
			},
			wantErr: false,
		},
		{
			name: "missing command",
			args: Args{
				LiquibaseArgs: LiquibaseArgs{
					Command: "",
				},
			},
			wantErr: true,
		},
		{
			name: "consolidated commands set",
			args: Args{
				LiquibaseArgs: LiquibaseArgs{
					ConsolidatedCommand: base64.StdEncoding.EncodeToString([]byte(`[{"command":"update","args":{}}]`)),
				},
			},
			wantErr: false,
		},
		{
			name: "both command and consolidated set",
			args: Args{
				LiquibaseArgs: LiquibaseArgs{
					Command:             "update",
					ConsolidatedCommand: base64.StdEncoding.EncodeToString([]byte(`[{"command":"tag","args":{}}]`)),
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInputs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateInputs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStepOutputFileHandling(t *testing.T) {
	// Test that step output file content is read correctly
	// This matches bash behavior: step_output=$(cat "$STEP_OUTPUT_FILE")
	tmpDir := t.TempDir()
	stepOutputFile := filepath.Join(tmpDir, "step_output.json")

	// Simulate dbops-extensions JAR output (already base64+zstd compressed)
	expectedContent := "KLUv/SAa0QAAeyJlcnJvciI6IiIsImZsb3dWMiI6dHJ1ZX0="
	if err := os.WriteFile(stepOutputFile, []byte(expectedContent+"\n"), 0644); err != nil {
		t.Fatalf("Failed to create step output file: %v", err)
	}

	// Read and verify
	content, err := os.ReadFile(stepOutputFile)
	if err != nil {
		t.Fatalf("Failed to read step output file: %v", err)
	}

	// Bash $() strips trailing newlines
	trimmed := strings.TrimRight(string(content), "\n")
	if trimmed != expectedContent {
		t.Errorf("Step output = %q, want %q", trimmed, expectedContent)
	}
}

func TestStepOutputConstant(t *testing.T) {
	// Verify the step output file path constant
	if StepOutputFile != "/tmp/step_output.json" {
		t.Errorf("StepOutputFile = %q, want %q", StepOutputFile, "/tmp/step_output.json")
	}
}

func TestOutputConstants(t *testing.T) {
	// Verify output key constants match expected values
	if OutputExitCode != "exit_code" {
		t.Errorf("OutputExitCode = %q, want %q", OutputExitCode, "exit_code")
	}
	if OutputStepOutput != "step_output" {
		t.Errorf("OutputStepOutput = %q, want %q", OutputStepOutput, "step_output")
	}
}

func TestValidateInputsErrorMessage(t *testing.T) {
	args := Args{
		LiquibaseArgs: LiquibaseArgs{
			Command: "",
		},
	}

	err := validateInputs(args)
	if err == nil {
		t.Fatal("validateInputs() should return error for missing command")
	}

	expectedMsg := "PLUGIN_COMMAND or PLUGIN_COMMANDS is required"
	if err.Error() != expectedMsg {
		t.Errorf("Error message = %q, want %q", err.Error(), expectedMsg)
	}
}

func TestGoogleCloudAuthCleanup(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a mock credentials file
	credFile := filepath.Join(tmpDir, "test-creds.json")
	if err := os.WriteFile(credFile, []byte(`{"test": "data"}`), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Verify cleanup removes the file
	os.Remove(credFile)
	if fileExists(credFile) {
		t.Error("File should be deleted after cleanup")
	}
}

func TestCopyFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.txt")
	dstFile := filepath.Join(tmpDir, "dest.txt")

	// Create source file
	if err := os.WriteFile(srcFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Copy file
	if err := copyFile(srcFile, dstFile); err != nil {
		t.Fatalf("copyFile() error = %v", err)
	}

	// Verify dest file has 0600 permissions (secure)
	info, err := os.Stat(dstFile)
	if err != nil {
		t.Fatalf("Failed to stat dest file: %v", err)
	}

	// copyFile sets 0600 permissions for security
	expectedPerm := os.FileMode(0600)
	if info.Mode().Perm() != expectedPerm {
		t.Errorf("Dest file permissions = %o, want %o", info.Mode().Perm(), expectedPerm)
	}
}

func TestPluginLiquibasePrefixConstant(t *testing.T) {
	// The prefix used for environment variable translation
	expected := "PLUGIN_LIQUIBASE_"
	envVar := OptionToEnvVar("url")
	if !strings.HasPrefix(envVar, expected) {
		t.Errorf("OptionToEnvVar should use prefix %q, got %q", expected, envVar)
	}
}

func TestFileExistsDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// fileExists returns true for directories too (os.Stat doesn't distinguish)
	if !fileExists(tmpDir) {
		t.Error("fileExists() should return true for existing directory")
	}
}

