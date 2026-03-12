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

func TestNewCommandBuilder(t *testing.T) {
	options := []string{"log-level", "search-path"}
	builder := NewCommandBuilder(options)

	if builder == nil {
		t.Fatal("NewCommandBuilder() returned nil")
	}
	if len(builder.globalOptions) != 2 {
		t.Errorf("globalOptions length = %d, want 2", len(builder.globalOptions))
	}
	if builder.processedVars == nil {
		t.Error("processedVars should be initialized")
	}
}

func TestBuildArgsBasicCommand(t *testing.T) {
	// Clear any existing PLUGIN_LIQUIBASE_* env vars
	for _, env := range os.Environ() {
		if len(env) > 17 && env[:17] == "PLUGIN_LIQUIBASE_" {
			key := env[:len(env)-len(env[len("PLUGIN_LIQUIBASE_"):])-1]
			os.Unsetenv(key)
		}
	}

	builder := NewCommandBuilder([]string{})
	args := Args{
		LiquibaseArgs: LiquibaseArgs{
			Command: "status",
		},
	}

	result, err := builder.BuildArgs(args)
	if err != nil {
		t.Fatalf("BuildArgs() error = %v", err)
	}

	if len(result) != 1 {
		t.Errorf("BuildArgs() returned %d args, want 1", len(result))
	}
	if result[0] != "status" {
		t.Errorf("BuildArgs()[0] = %q, want %q", result[0], "status")
	}
}

func TestBuildArgsWithGlobalOptions(t *testing.T) {
	// Set environment variable
	os.Setenv("PLUGIN_LIQUIBASE_LOG_LEVEL", "debug")
	defer os.Unsetenv("PLUGIN_LIQUIBASE_LOG_LEVEL")

	builder := NewCommandBuilder([]string{"log-level"})
	args := Args{
		LiquibaseArgs: LiquibaseArgs{
			Command: "update",
		},
	}

	result, err := builder.BuildArgs(args)
	if err != nil {
		t.Fatalf("BuildArgs() error = %v", err)
	}

	// Should have: --log-level debug update
	if len(result) < 3 {
		t.Fatalf("BuildArgs() returned %d args, want at least 3", len(result))
	}

	if result[0] != "--log-level" {
		t.Errorf("result[0] = %q, want %q", result[0], "--log-level")
	}
	if result[1] != "debug" {
		t.Errorf("result[1] = %q, want %q", result[1], "debug")
	}
	if result[2] != "update" {
		t.Errorf("result[2] = %q, want %q", result[2], "update")
	}
}

func TestProcessRemainingEnvVars(t *testing.T) {
	// Set a PLUGIN_LIQUIBASE_* environment variable
	os.Setenv("PLUGIN_LIQUIBASE_CHANGELOG_FILE", "test.xml")
	defer os.Unsetenv("PLUGIN_LIQUIBASE_CHANGELOG_FILE")

	builder := NewCommandBuilder([]string{}) // No global options
	args := builder.processRemainingEnvVars()

	found := false
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--changelog-file" && args[i+1] == "test.xml" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("processRemainingEnvVars() did not include --changelog-file test.xml, got: %v", args)
	}
}
