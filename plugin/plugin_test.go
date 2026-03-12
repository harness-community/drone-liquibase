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

func TestOptionToEnvVar(t *testing.T) {
	tests := []struct {
		option   string
		expected string
	}{
		{"log-level", "PLUGIN_LIQUIBASE_LOG_LEVEL"},
		{"search-path", "PLUGIN_LIQUIBASE_SEARCH_PATH"},
		{"changelog-file", "PLUGIN_LIQUIBASE_CHANGELOG_FILE"},
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
