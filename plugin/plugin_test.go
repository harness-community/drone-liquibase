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
