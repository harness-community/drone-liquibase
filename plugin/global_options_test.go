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

func TestLoadGlobalOptionsEmptyLines(t *testing.T) {
	tmpDir := t.TempDir()
	optionsFile := filepath.Join(tmpDir, "options.txt")

	// Content with empty lines and whitespace
	content := "log-level\n\n  \nchangelog-file\n\n"
	if err := os.WriteFile(optionsFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	options, err := LoadGlobalOptions(optionsFile)
	if err != nil {
		t.Fatalf("LoadGlobalOptions() error = %v", err)
	}

	// Should only have 2 options, empty lines filtered
	if len(options) != 2 {
		t.Errorf("Got %d options, want 2 (empty lines should be filtered)", len(options))
	}
}

func TestLoadGlobalOptionsWhitespace(t *testing.T) {
	tmpDir := t.TempDir()
	optionsFile := filepath.Join(tmpDir, "options.txt")

	// Content with leading/trailing whitespace
	content := "  log-level  \nchangelog-file\n"
	if err := os.WriteFile(optionsFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	options, err := LoadGlobalOptions(optionsFile)
	if err != nil {
		t.Fatalf("LoadGlobalOptions() error = %v", err)
	}

	// First option should be trimmed
	if options[0] != "log-level" {
		t.Errorf("options[0] = %q, want %q (should be trimmed)", options[0], "log-level")
	}
}

func TestOptionToEnvVarConversions(t *testing.T) {
	tests := []struct {
		option   string
		expected string
	}{
		{"log-level", "PLUGIN_LIQUIBASE_LOG_LEVEL"},
		{"search-path", "PLUGIN_LIQUIBASE_SEARCH_PATH"},
		{"changelog-file", "PLUGIN_LIQUIBASE_CHANGELOG_FILE"},
		{"url", "PLUGIN_LIQUIBASE_URL"},
		{"default-schema-name", "PLUGIN_LIQUIBASE_DEFAULT_SCHEMA_NAME"},
		{"driver-properties-file", "PLUGIN_LIQUIBASE_DRIVER_PROPERTIES_FILE"},
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

func TestEnvVarToOptionConversions(t *testing.T) {
	tests := []struct {
		envVar   string
		expected string
	}{
		{"LOG_LEVEL", "log-level"},
		{"SEARCH_PATH", "search-path"},
		{"CHANGELOG_FILE", "changelog-file"},
		{"URL", "url"},
		{"DEFAULT_SCHEMA_NAME", "default-schema-name"},
		{"DRIVER_PROPERTIES_FILE", "driver-properties-file"},
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

func TestRoundTripConversions(t *testing.T) {
	// Converting option -> env var -> option should give back original
	options := []string{
		"log-level",
		"search-path",
		"changelog-file",
		"default-catalog-name",
	}

	for _, opt := range options {
		t.Run(opt, func(t *testing.T) {
			envVar := OptionToEnvVar(opt)
			// Extract the suffix (remove PLUGIN_LIQUIBASE_ prefix)
			suffix := envVar[17:] // len("PLUGIN_LIQUIBASE_") = 17
			backToOption := EnvVarToOption(suffix)

			if backToOption != opt {
				t.Errorf("Round trip failed: %q -> %q -> %q", opt, envVar, backToOption)
			}
		})
	}
}
