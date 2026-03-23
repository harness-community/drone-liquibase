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
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/harness/liquibase-drone-plugin/internal/execution"
)

// mockRunCommandSuccess simulates a command that always succeeds.
func mockRunCommandSuccess(name string, args ...string) (int, []byte, error) {
	return 0, nil, nil
}

// mockRunCommandFailure simulates a command that always fails with exit code 1.
func mockRunCommandFailure(name string, args ...string) (int, []byte, error) {
	return 1, nil, fmt.Errorf("command failed")
}

func TestDecodeCommands(t *testing.T) {
	commands := []ConsolidatedCommand{
		{
			Command: "update",
			Args: map[string]string{
				"PLUGIN_LIQUIBASE_CHANGELOG_FILE": "changelog.xml",
				"PLUGIN_LIQUIBASE_URL":            "jdbc:postgresql://localhost:5432/testdb",
			},
		},
		{
			Command: "tag",
			Args: map[string]string{
				"PLUGIN_LIQUIBASE_TAG": "v1.0",
			},
		},
	}

	jsonBytes, err := json.Marshal(commands)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}
	encoded := base64.StdEncoding.EncodeToString(jsonBytes)

	result, err := decodeCommands(encoded)
	if err != nil {
		t.Fatalf("decodeCommands() error = %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("decodeCommands() returned %d commands, want 2", len(result))
	}

	if result[0].Command != "update" {
		t.Errorf("result[0].Command = %q, want %q", result[0].Command, "update")
	}
	if result[0].Args["PLUGIN_LIQUIBASE_CHANGELOG_FILE"] != "changelog.xml" {
		t.Errorf("result[0].Args[PLUGIN_LIQUIBASE_CHANGELOG_FILE] = %q, want %q", result[0].Args["PLUGIN_LIQUIBASE_CHANGELOG_FILE"], "changelog.xml")
	}
	if result[1].Command != "tag" {
		t.Errorf("result[1].Command = %q, want %q", result[1].Command, "tag")
	}
	if result[1].Args["PLUGIN_LIQUIBASE_TAG"] != "v1.0" {
		t.Errorf("result[1].Args[PLUGIN_LIQUIBASE_TAG] = %q, want %q", result[1].Args["PLUGIN_LIQUIBASE_TAG"], "v1.0")
	}
}

func TestDecodeCommandsInvalidBase64(t *testing.T) {
	_, err := decodeCommands("not-valid-base64!!!")
	if err == nil {
		t.Error("decodeCommands() should error for invalid base64")
	}
}

func TestDecodeCommandsInvalidJSON(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("not json"))
	_, err := decodeCommands(encoded)
	if err == nil {
		t.Error("decodeCommands() should error for invalid JSON")
	}
}

func TestDecodeCommandsEmpty(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("[]"))
	result, err := decodeCommands(encoded)
	if err != nil {
		t.Fatalf("decodeCommands() error = %v", err)
	}
	if len(result) != 0 {
		t.Errorf("decodeCommands() returned %d commands, want 0", len(result))
	}
}

func TestConsolidatedCommandJSON(t *testing.T) {
	cmd := ConsolidatedCommand{
		Command: "update",
		Args: map[string]string{
			"PLUGIN_LIQUIBASE_URL":      "jdbc:postgresql://localhost/db",
			"PLUGIN_LIQUIBASE_USERNAME": "user",
		},
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded ConsolidatedCommand
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Command != cmd.Command {
		t.Errorf("Command = %q, want %q", decoded.Command, cmd.Command)
	}
	if decoded.Args["PLUGIN_LIQUIBASE_URL"] != cmd.Args["PLUGIN_LIQUIBASE_URL"] {
		t.Errorf("Args[PLUGIN_LIQUIBASE_URL] = %q, want %q", decoded.Args["PLUGIN_LIQUIBASE_URL"], cmd.Args["PLUGIN_LIQUIBASE_URL"])
	}
	if decoded.Args["PLUGIN_LIQUIBASE_USERNAME"] != cmd.Args["PLUGIN_LIQUIBASE_USERNAME"] {
		t.Errorf("Args[PLUGIN_LIQUIBASE_USERNAME] = %q, want %q", decoded.Args["PLUGIN_LIQUIBASE_USERNAME"], cmd.Args["PLUGIN_LIQUIBASE_USERNAME"])
	}
}

func TestDecodeCommandsSingleCommand(t *testing.T) {
	commands := []ConsolidatedCommand{
		{Command: "status", Args: map[string]string{"PLUGIN_LIQUIBASE_URL": "jdbc:h2:mem:test"}},
	}
	jsonBytes, _ := json.Marshal(commands)
	encoded := base64.StdEncoding.EncodeToString(jsonBytes)

	result, err := decodeCommands(encoded)
	if err != nil {
		t.Fatalf("decodeCommands() error = %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("decodeCommands() returned %d commands, want 1", len(result))
	}
	if result[0].Command != "status" {
		t.Errorf("Command = %q, want %q", result[0].Command, "status")
	}
}

func TestDecodeCommandsWrongJSONStructure(t *testing.T) {
	// JSON object instead of array
	encoded := base64.StdEncoding.EncodeToString([]byte(`{"command":"update"}`))
	_, err := decodeCommands(encoded)
	if err == nil {
		t.Error("decodeCommands() should error for JSON object instead of array")
	}
}

func TestExecuteConsolidatedEnvVarCleanup(t *testing.T) {
	origRunCmd := runCommandWithOutput
	runCommandWithOutput = mockRunCommandSuccess
	defer func() { runCommandWithOutput = origRunCmd }()

	// Verify that env vars set for one command are cleaned up after execution
	commands := []ConsolidatedCommand{
		{
			Command: "hello",
			Args:    map[string]string{"PLUGIN_LIQUIBASE_URL": "jdbc:test://db1", "PLUGIN_LIQUIBASE_USERNAME": "testuser"},
		},
	}

	pluginOutput := execution.NewOutput()
	args := Args{}

	err := executeConsolidated(args, []string{}, commands, pluginOutput)
	if err != nil {
		t.Fatalf("executeConsolidated() error = %v", err)
	}

	// Verify env vars from the command args were cleaned up
	for key := range commands[0].Args {
		if val := os.Getenv(key); val != "" {
			t.Errorf("Env var %s should be unset after executeConsolidated, got %q", key, val)
		}
	}
}

func TestExecuteConsolidatedSuccess(t *testing.T) {
	origRunCmd := runCommandWithOutput
	runCommandWithOutput = mockRunCommandSuccess
	defer func() { runCommandWithOutput = origRunCmd }()

	commands := []ConsolidatedCommand{
		{Command: "update", Args: map[string]string{}},
	}

	pluginOutput := execution.NewOutput()
	args := Args{}

	err := executeConsolidated(args, []string{}, commands, pluginOutput)
	if err != nil {
		t.Fatalf("executeConsolidated() error = %v", err)
	}
}

func TestExecuteConsolidatedFailure(t *testing.T) {
	origRunCmd := runCommandWithOutput
	runCommandWithOutput = mockRunCommandFailure
	defer func() { runCommandWithOutput = origRunCmd }()

	commands := []ConsolidatedCommand{
		{Command: "update", Args: map[string]string{}},
	}

	pluginOutput := execution.NewOutput()
	args := Args{}

	err := executeConsolidated(args, []string{}, commands, pluginOutput)
	if err == nil {
		t.Error("executeConsolidated() should error on command failure")
	}
}

func TestExecuteConsolidatedStructFieldSync(t *testing.T) {
	origRunCmd := runCommandWithOutput
	runCommandWithOutput = mockRunCommandSuccess
	defer func() { runCommandWithOutput = origRunCmd }()

	// Verify that PLUGIN_SUBSTITUTE_LIQUIBASE and GENERATE_STEP_OUTPUTS
	// from cmd.Args are synced to the struct fields used by BuildArgs
	commands := []ConsolidatedCommand{
		{
			Command: "hello",
			Args: map[string]string{
				"PLUGIN_SUBSTITUTE_LIQUIBASE": "some-encoded-value",
				"GENERATE_STEP_OUTPUTS":      "true",
			},
		},
	}

	pluginOutput := execution.NewOutput()
	args := Args{}

	// SubstituteLiquibase will cause BuildArgs to fail (invalid base64+zstd),
	// but that confirms the value was synced to the struct field
	err := executeConsolidated(args, []string{}, commands, pluginOutput)
	if err == nil {
		t.Error("executeConsolidated() should error due to invalid PLUGIN_SUBSTITUTE_LIQUIBASE")
	}
	if !strings.Contains(err.Error(), "substitution properties") {
		t.Errorf("error should mention substitution properties, got: %v", err)
	}
}

func TestExecuteConsolidatedMultiCommandFailsOnSecond(t *testing.T) {
	callCount := 0
	origRunCmd := runCommandWithOutput
	// First command succeeds, second command fails
	runCommandWithOutput = func(name string, args ...string) (int, []byte, error) {
		callCount++
		if callCount == 1 {
			return 0, nil, nil
		}
		return 1, nil, fmt.Errorf("command failed")
	}
	defer func() { runCommandWithOutput = origRunCmd }()

	commands := []ConsolidatedCommand{
		{Command: "first", Args: map[string]string{}},
		{Command: "second", Args: map[string]string{}},
	}

	pluginOutput := execution.NewOutput()
	args := Args{}

	err := executeConsolidated(args, []string{}, commands, pluginOutput)
	if err == nil {
		t.Fatal("executeConsolidated() should error on command failure")
	}
	// Should fail on command 2
	if !strings.Contains(err.Error(), "command 2") {
		t.Errorf("error should reference command 2, got: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 command executions, got %d", callCount)
	}
}

func TestExecuteConsolidatedEnvVarIsolationBetweenCommands(t *testing.T) {
	origRunCmd := runCommandWithOutput
	// Track env vars at the time each command executes
	var cmd1URL, cmd1Username, cmd2URL, cmd2Username string
	callCount := 0
	runCommandWithOutput = func(name string, args ...string) (int, []byte, error) {
		callCount++
		if callCount == 1 {
			cmd1URL = os.Getenv("PLUGIN_LIQUIBASE_URL")
			cmd1Username = os.Getenv("PLUGIN_LIQUIBASE_USERNAME")
		} else {
			cmd2URL = os.Getenv("PLUGIN_LIQUIBASE_URL")
			cmd2Username = os.Getenv("PLUGIN_LIQUIBASE_USERNAME")
		}
		return 0, nil, nil
	}
	defer func() { runCommandWithOutput = origRunCmd }()

	commands := []ConsolidatedCommand{
		{
			Command: "first",
			Args:    map[string]string{"PLUGIN_LIQUIBASE_URL": "jdbc:first://db"},
		},
		{
			Command: "second",
			Args:    map[string]string{"PLUGIN_LIQUIBASE_USERNAME": "user2"},
		},
	}

	pluginOutput := execution.NewOutput()
	args := Args{}

	err := executeConsolidated(args, []string{}, commands, pluginOutput)
	if err != nil {
		t.Fatalf("executeConsolidated() error = %v", err)
	}

	// Command 1 should see its own URL but not command 2's username
	if cmd1URL != "jdbc:first://db" {
		t.Errorf("command 1: PLUGIN_LIQUIBASE_URL = %q, want %q", cmd1URL, "jdbc:first://db")
	}
	if cmd1Username != "" {
		t.Errorf("command 1: PLUGIN_LIQUIBASE_USERNAME should be empty, got %q", cmd1Username)
	}

	// Command 2 should see its own username but NOT command 1's URL (cleaned up)
	if cmd2URL != "" {
		t.Errorf("command 2: PLUGIN_LIQUIBASE_URL should be empty (cleaned up from cmd 1), got %q", cmd2URL)
	}
	if cmd2Username != "user2" {
		t.Errorf("command 2: PLUGIN_LIQUIBASE_USERNAME = %q, want %q", cmd2Username, "user2")
	}

	// After execution, all env vars from both commands should be cleaned up
	if val := os.Getenv("PLUGIN_LIQUIBASE_URL"); val != "" {
		t.Errorf("PLUGIN_LIQUIBASE_URL should be unset, got %q", val)
	}
	if val := os.Getenv("PLUGIN_LIQUIBASE_USERNAME"); val != "" {
		t.Errorf("PLUGIN_LIQUIBASE_USERNAME should be unset, got %q", val)
	}
}

func TestConsolidatedCommandNoArgs(t *testing.T) {
	cmd := ConsolidatedCommand{
		Command: "status",
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded ConsolidatedCommand
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Command != "status" {
		t.Errorf("Command = %q, want %q", decoded.Command, "status")
	}
	if decoded.Args != nil {
		t.Errorf("Args should be nil, got %v", decoded.Args)
	}
}
