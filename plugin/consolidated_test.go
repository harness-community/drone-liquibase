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
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/harness/liquibase-drone-plugin/internal/execution"
)

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
		t.Errorf("result[0].Args[PLUGIN_LIQUIBASE_CHANGELOG_FILE] = %q, want %q",
			result[0].Args["PLUGIN_LIQUIBASE_CHANGELOG_FILE"], "changelog.xml")
	}
	if result[1].Command != "tag" {
		t.Errorf("result[1].Command = %q, want %q", result[1].Command, "tag")
	}
	if result[1].Args["PLUGIN_LIQUIBASE_TAG"] != "v1.0" {
		t.Errorf("result[1].Args[PLUGIN_LIQUIBASE_TAG] = %q, want %q",
			result[1].Args["PLUGIN_LIQUIBASE_TAG"], "v1.0")
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
	encoded := base64.StdEncoding.EncodeToString([]byte(`{"command":"update"}`))
	_, err := decodeCommands(encoded)
	if err == nil {
		t.Error("decodeCommands() should error for JSON object instead of array")
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
		t.Errorf("Args[PLUGIN_LIQUIBASE_URL] = %q, want %q",
			decoded.Args["PLUGIN_LIQUIBASE_URL"], cmd.Args["PLUGIN_LIQUIBASE_URL"])
	}
}

func TestConsolidatedCommandNoArgs(t *testing.T) {
	cmd := ConsolidatedCommand{Command: "status"}

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

func TestRunnerCommandJSON(t *testing.T) {
	cmd := RunnerCommand{
		CliArgs: []string{"--url=jdbc:h2:mem:test", "update"},
		EnvVars: map[string]string{"PLUGIN_BEARER_TOKEN": "secret"},
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded RunnerCommand
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(decoded.CliArgs) != 2 || decoded.CliArgs[0] != "--url=jdbc:h2:mem:test" {
		t.Errorf("CliArgs = %v, want [--url=jdbc:h2:mem:test update]", decoded.CliArgs)
	}
	if decoded.EnvVars["PLUGIN_BEARER_TOKEN"] != "secret" {
		t.Errorf("EnvVars[PLUGIN_BEARER_TOKEN] = %q, want %q",
			decoded.EnvVars["PLUGIN_BEARER_TOKEN"], "secret")
	}
}

func TestRunnerCommandJSONOmitsEmptyEnvVars(t *testing.T) {
	cmd := RunnerCommand{CliArgs: []string{"status"}}

	data, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	if strings.Contains(string(data), "envVars") {
		t.Errorf("JSON should omit envVars when nil, got: %s", string(data))
	}
}

func TestRunnerResultJSON(t *testing.T) {
	// Test deserialization matching what CommandRunner.java writes
	data := []byte(`{"exitCode":0}`)
	var result RunnerResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}

	data = []byte(`{"exitCode":1}`)
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", result.ExitCode)
	}
}

func TestSetEnvVars(t *testing.T) {
	vars := map[string]string{
		"TEST_CONSOLIDATED_A": "valueA",
		"TEST_CONSOLIDATED_B": "valueB",
	}
	defer unsetEnvVars([]string{"TEST_CONSOLIDATED_A", "TEST_CONSOLIDATED_B"})

	keys := setEnvVars(vars)

	if len(keys) != 2 {
		t.Fatalf("setEnvVars returned %d keys, want 2", len(keys))
	}
	if os.Getenv("TEST_CONSOLIDATED_A") != "valueA" {
		t.Errorf("TEST_CONSOLIDATED_A = %q, want %q", os.Getenv("TEST_CONSOLIDATED_A"), "valueA")
	}
	if os.Getenv("TEST_CONSOLIDATED_B") != "valueB" {
		t.Errorf("TEST_CONSOLIDATED_B = %q, want %q", os.Getenv("TEST_CONSOLIDATED_B"), "valueB")
	}
}

func TestUnsetEnvVars(t *testing.T) {
	os.Setenv("TEST_CONSOLIDATED_UNSET", "should-be-removed")
	unsetEnvVars([]string{"TEST_CONSOLIDATED_UNSET"})

	if val := os.Getenv("TEST_CONSOLIDATED_UNSET"); val != "" {
		t.Errorf("TEST_CONSOLIDATED_UNSET should be unset, got %q", val)
	}
}

func TestSetEnvVarsEmpty(t *testing.T) {
	keys := setEnvVars(map[string]string{})
	if len(keys) != 0 {
		t.Errorf("setEnvVars({}) returned %d keys, want 0", len(keys))
	}
}

func TestSetUnsetEnvVarsRoundTrip(t *testing.T) {
	vars := map[string]string{
		"TEST_ROUNDTRIP_X": "x",
		"TEST_ROUNDTRIP_Y": "y",
	}

	keys := setEnvVars(vars)
	// Verify set
	for k, v := range vars {
		if got := os.Getenv(k); got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}

	unsetEnvVars(keys)
	// Verify unset
	for k := range vars {
		if got := os.Getenv(k); got != "" {
			t.Errorf("%s should be unset after cleanup, got %q", k, got)
		}
	}
}

func TestWaitForRunnerResultSuccess(t *testing.T) {
	// Clean up any leftover file
	os.Remove(RunnerResultFile)
	defer os.Remove(RunnerResultFile)

	jvmDone := make(chan error, 1)

	// Write result file before calling waitForRunnerResult
	if err := os.WriteFile(RunnerResultFile, []byte(`{"exitCode":0}`), 0644); err != nil {
		t.Fatalf("Failed to write result file: %v", err)
	}

	result, err := waitForRunnerResult(jvmDone)
	if err != nil {
		t.Fatalf("waitForRunnerResult() error = %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
}

func TestWaitForRunnerResultNonZeroExit(t *testing.T) {
	os.Remove(RunnerResultFile)
	defer os.Remove(RunnerResultFile)

	jvmDone := make(chan error, 1)

	if err := os.WriteFile(RunnerResultFile, []byte(`{"exitCode":1}`), 0644); err != nil {
		t.Fatalf("Failed to write result file: %v", err)
	}

	result, err := waitForRunnerResult(jvmDone)
	if err != nil {
		t.Fatalf("waitForRunnerResult() error = %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", result.ExitCode)
	}
}

func TestWaitForRunnerResultJVMCrash(t *testing.T) {
	os.Remove(RunnerResultFile)

	jvmDone := make(chan error, 1)
	jvmDone <- fmt.Errorf("exit status 1")

	_, err := waitForRunnerResult(jvmDone)
	if err == nil {
		t.Fatal("waitForRunnerResult() should error on JVM crash")
	}
	if !strings.Contains(err.Error(), "JVM process crashed") {
		t.Errorf("error should mention JVM crash, got: %v", err)
	}
}

func TestWaitForRunnerResultJVMExitNoError(t *testing.T) {
	os.Remove(RunnerResultFile)

	jvmDone := make(chan error, 1)
	jvmDone <- nil // JVM exited cleanly but unexpectedly

	_, err := waitForRunnerResult(jvmDone)
	if err == nil {
		t.Fatal("waitForRunnerResult() should error on unexpected JVM exit")
	}
	if !strings.Contains(err.Error(), "exited unexpectedly") {
		t.Errorf("error should mention unexpected exit, got: %v", err)
	}
}

func TestWaitForRunnerResultInvalidJSON(t *testing.T) {
	os.Remove(RunnerResultFile)
	defer os.Remove(RunnerResultFile)

	jvmDone := make(chan error, 1)

	if err := os.WriteFile(RunnerResultFile, []byte(`not json`), 0644); err != nil {
		t.Fatalf("Failed to write result file: %v", err)
	}

	_, err := waitForRunnerResult(jvmDone)
	if err == nil {
		t.Fatal("waitForRunnerResult() should error on invalid JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse runner result") {
		t.Errorf("error should mention parse failure, got: %v", err)
	}
}

func TestWaitForRunnerResultCleansUpFile(t *testing.T) {
	os.Remove(RunnerResultFile)
	defer os.Remove(RunnerResultFile)

	jvmDone := make(chan error, 1)

	if err := os.WriteFile(RunnerResultFile, []byte(`{"exitCode":0}`), 0644); err != nil {
		t.Fatalf("Failed to write result file: %v", err)
	}

	_, err := waitForRunnerResult(jvmDone)
	if err != nil {
		t.Fatalf("waitForRunnerResult() error = %v", err)
	}

	if fileExists(RunnerResultFile) {
		t.Error("RunnerResultFile should be removed after reading")
	}
}

func TestSendCommandWritesJSONToStdin(t *testing.T) {
	os.Remove(RunnerResultFile)
	defer os.Remove(RunnerResultFile)

	var buf bytes.Buffer
	jvmDone := make(chan error, 1)

	cliArgs := []string{"--url=jdbc:h2:mem:test", "update"}
	envVars := map[string]string{"PLUGIN_BEARER_TOKEN": "tok"}

	// sendCommand removes RunnerResultFile then polls for it every 500ms,
	// so write it after a short delay to simulate the Java runner.
	go func() {
		time.Sleep(100 * time.Millisecond)
		os.WriteFile(RunnerResultFile, []byte(`{"exitCode":0}`), 0644)
	}()

	exitCode, err := sendCommand(&buf, jvmDone, cliArgs, envVars)
	if err != nil {
		t.Fatalf("sendCommand() error = %v", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}

	// Verify the JSON written to stdin (first line before the newline)
	written := strings.TrimSpace(buf.String())
	var cmd RunnerCommand
	if err := json.Unmarshal([]byte(written), &cmd); err != nil {
		t.Fatalf("Failed to parse stdin JSON: %v (written: %q)", err, written)
	}
	if len(cmd.CliArgs) != 2 || cmd.CliArgs[0] != "--url=jdbc:h2:mem:test" {
		t.Errorf("CliArgs = %v, want [--url=jdbc:h2:mem:test update]", cmd.CliArgs)
	}
	if cmd.EnvVars["PLUGIN_BEARER_TOKEN"] != "tok" {
		t.Errorf("EnvVars[PLUGIN_BEARER_TOKEN] = %q, want %q", cmd.EnvVars["PLUGIN_BEARER_TOKEN"], "tok")
	}
}

func TestSendCommandCleansUpPreviousFiles(t *testing.T) {
	// Create stale files from a "previous command"
	os.WriteFile(RunnerResultFile, []byte(`{"exitCode":99}`), 0644)
	os.WriteFile(StepOutputFile, []byte(`old output`), 0644)
	defer os.Remove(RunnerResultFile)
	defer os.Remove(StepOutputFile)

	var buf bytes.Buffer
	jvmDone := make(chan error, 1)

	// Use a channel to verify files are removed before the result is written
	cleaned := make(chan bool, 1)
	go func() {
		// Wait for sendCommand to write to stdin (it removes files before writing)
		time.Sleep(100 * time.Millisecond)
		resultGone := !fileExists(RunnerResultFile)
		stepOutputGone := !fileExists(StepOutputFile)
		cleaned <- resultGone && stepOutputGone
		// Now write the result so sendCommand can return
		os.WriteFile(RunnerResultFile, []byte(`{"exitCode":0}`), 0644)
	}()

	exitCode, err := sendCommand(&buf, jvmDone, []string{"status"}, nil)
	if err != nil {
		t.Fatalf("sendCommand() error = %v", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}

	if wasCleaned := <-cleaned; !wasCleaned {
		t.Error("sendCommand should remove RunnerResultFile and StepOutputFile before polling")
	}
}

func TestSendCommandNonZeroExitCode(t *testing.T) {
	os.Remove(RunnerResultFile)
	defer os.Remove(RunnerResultFile)

	var buf bytes.Buffer
	jvmDone := make(chan error, 1)

	go func() {
		time.Sleep(100 * time.Millisecond)
		os.WriteFile(RunnerResultFile, []byte(`{"exitCode":1}`), 0644)
	}()

	exitCode, err := sendCommand(&buf, jvmDone, []string{"update"}, nil)
	if err != nil {
		t.Fatalf("sendCommand() error = %v", err)
	}
	if exitCode != 1 {
		t.Errorf("exitCode = %d, want 1", exitCode)
	}
}

func TestConstants(t *testing.T) {
	if RunnerResultFile != "/tmp/runner_result.json" {
		t.Errorf("RunnerResultFile = %q, want /tmp/runner_result.json", RunnerResultFile)
	}
	if CommandRunnerClass != "com.harness.dbops.runner.CommandRunner" {
		t.Errorf("CommandRunnerClass = %q, want com.harness.dbops.runner.CommandRunner", CommandRunnerClass)
	}
	if !strings.Contains(runnerClasspath, "/liquibase/internal/lib/*") {
		t.Errorf("runnerClasspath should contain /liquibase/internal/lib/*, got %q", runnerClasspath)
	}
	if !strings.Contains(runnerClasspath, "/liquibase/lib/*") {
		t.Errorf("runnerClasspath should contain /liquibase/lib/*, got %q", runnerClasspath)
	}
	if !strings.Contains(runnerClasspath, "/liquibase/internal/extensions/*") {
		t.Errorf("runnerClasspath should contain /liquibase/internal/extensions/*, got %q", runnerClasspath)
	}
}

func TestBuildCommandArgsSetsAndUnsetsEnvVars(t *testing.T) {
	command := ConsolidatedCommand{
		Command: "update",
		Args: map[string]string{
			"PLUGIN_LIQUIBASE_URL":      "jdbc:test://db",
			"PLUGIN_LIQUIBASE_USERNAME": "testuser",
		},
	}

	// buildCommandArgs will fail because there's no real liquibase,
	// but we can verify env var cleanup
	_, _ = buildCommandArgs(Args{}, []string{}, command)

	// After buildCommandArgs returns (with defer unsetEnvVars), env vars should be cleaned up
	if val := os.Getenv("PLUGIN_LIQUIBASE_URL"); val != "" {
		t.Errorf("PLUGIN_LIQUIBASE_URL should be unset after buildCommandArgs, got %q", val)
	}
	if val := os.Getenv("PLUGIN_LIQUIBASE_USERNAME"); val != "" {
		t.Errorf("PLUGIN_LIQUIBASE_USERNAME should be unset after buildCommandArgs, got %q", val)
	}
}

func TestBuildCommandArgsSyncsSubstituteLiquibase(t *testing.T) {
	command := ConsolidatedCommand{
		Command: "update",
		Args: map[string]string{
			"PLUGIN_SUBSTITUTE_LIQUIBASE": "some-encoded-value",
		},
	}

	// SubstituteLiquibase will cause BuildArgs to fail (invalid base64+zstd),
	// but that confirms the value was synced to the struct field
	_, err := buildCommandArgs(Args{}, []string{}, command)
	if err == nil {
		t.Error("buildCommandArgs() should error due to invalid PLUGIN_SUBSTITUTE_LIQUIBASE")
	}
	if !strings.Contains(err.Error(), "substitution properties") {
		t.Errorf("error should mention substitution properties, got: %v", err)
	}
}

func TestBuildCommandArgsSyncsGenerateStepOutputs(t *testing.T) {
	command := ConsolidatedCommand{
		Command: "update",
		Args: map[string]string{
			"GENERATE_STEP_OUTPUTS": "true",
		},
	}

	// This tests that GENERATE_STEP_OUTPUTS is synced to the struct field.
	// BuildArgs should pick it up (it won't fail because of this field alone).
	_, _ = buildCommandArgs(Args{}, []string{}, command)
	// No error assertion — just verifying it doesn't panic
}

// nopWriteCloser wraps an io.Writer with a no-op Close method.
type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }

// fakeRunner simulates the Java CommandRunner. It reads JSON commands from
// the provided buffer and writes result files with the configured exit codes.
// Must be started as a goroutine — it polls the buffer for new commands.
func fakeRunner(buf *bytes.Buffer, exitCodes []int) <-chan error {
	jvmDone := make(chan error, 1)
	go func() {
		cmdIdx := 0
		for cmdIdx < len(exitCodes) {
			time.Sleep(50 * time.Millisecond)
			// Check if a command was written to stdin
			if buf.Len() == 0 {
				continue
			}
			line, err := buf.ReadString('\n')
			if err != nil || line == "" {
				continue
			}
			exitCode := exitCodes[cmdIdx]
			os.WriteFile(RunnerResultFile, []byte(fmt.Sprintf(`{"exitCode":%d}`, exitCode)), 0644)
			cmdIdx++
		}
	}()
	return jvmDone
}

func TestExecuteConsolidatedAllSuccess(t *testing.T) {
	os.Remove(RunnerResultFile)
	defer os.Remove(RunnerResultFile)

	var buf bytes.Buffer
	origStartRunner := startRunner
	startRunner = func() (io.WriteCloser, <-chan error, error) {
		jvmDone := fakeRunner(&buf, []int{0, 0})
		return nopWriteCloser{&buf}, jvmDone, nil
	}
	defer func() { startRunner = origStartRunner }()

	commands := []ConsolidatedCommand{
		{Command: "update", Args: map[string]string{"PLUGIN_LIQUIBASE_URL": "jdbc:h2:mem:test"}},
		{Command: "tag", Args: map[string]string{"PLUGIN_LIQUIBASE_TAG": "v1.0"}},
	}

	pluginOutput := execution.NewOutput()
	err := executeConsolidated(Args{}, []string{}, commands, pluginOutput)
	if err != nil {
		t.Fatalf("executeConsolidated() error = %v", err)
	}

	// Last exit code should be 0
	exitCode := pluginOutput.GetProperty(OutputExitCode)
	if exitCode != "0" {
		t.Errorf("exit_code = %v, want \"0\"", exitCode)
	}
}

func TestExecuteConsolidatedStopsOnFailure(t *testing.T) {
	os.Remove(RunnerResultFile)
	defer os.Remove(RunnerResultFile)

	var buf bytes.Buffer
	origStartRunner := startRunner
	startRunner = func() (io.WriteCloser, <-chan error, error) {
		// First command succeeds, second fails
		jvmDone := fakeRunner(&buf, []int{0, 1})
		return nopWriteCloser{&buf}, jvmDone, nil
	}
	defer func() { startRunner = origStartRunner }()

	commands := []ConsolidatedCommand{
		{Command: "update", Args: map[string]string{"PLUGIN_LIQUIBASE_URL": "jdbc:h2:mem:test"}},
		{Command: "tag", Args: map[string]string{"PLUGIN_LIQUIBASE_TAG": "v1.0"}},
	}

	pluginOutput := execution.NewOutput()
	err := executeConsolidated(Args{}, []string{}, commands, pluginOutput)
	// Should return nil even on failure (error handled via exit code)
	if err != nil {
		t.Fatalf("executeConsolidated() error = %v, want nil", err)
	}

	exitCode := pluginOutput.GetProperty(OutputExitCode)
	if exitCode != "1" {
		t.Errorf("exit_code = %v, want \"1\"", exitCode)
	}
}

func TestExecuteConsolidatedStartRunnerError(t *testing.T) {
	origStartRunner := startRunner
	startRunner = func() (io.WriteCloser, <-chan error, error) {
		return nil, nil, fmt.Errorf("JVM failed to start")
	}
	defer func() { startRunner = origStartRunner }()

	pluginOutput := execution.NewOutput()
	err := executeConsolidated(Args{}, []string{}, []ConsolidatedCommand{{Command: "update"}}, pluginOutput)
	if err == nil {
		t.Fatal("executeConsolidated() should return error when startRunner fails")
	}
	if !strings.Contains(err.Error(), "JVM failed to start") {
		t.Errorf("error = %v, want 'JVM failed to start'", err)
	}
}

func TestExecuteConsolidatedSingleCommand(t *testing.T) {
	os.Remove(RunnerResultFile)
	defer os.Remove(RunnerResultFile)

	var buf bytes.Buffer
	origStartRunner := startRunner
	startRunner = func() (io.WriteCloser, <-chan error, error) {
		jvmDone := fakeRunner(&buf, []int{0})
		return nopWriteCloser{&buf}, jvmDone, nil
	}
	defer func() { startRunner = origStartRunner }()

	commands := []ConsolidatedCommand{
		{Command: "status", Args: map[string]string{"PLUGIN_LIQUIBASE_URL": "jdbc:h2:mem:test"}},
	}

	pluginOutput := execution.NewOutput()
	err := executeConsolidated(Args{}, []string{}, commands, pluginOutput)
	if err != nil {
		t.Fatalf("executeConsolidated() error = %v", err)
	}

	exitCode := pluginOutput.GetProperty(OutputExitCode)
	if exitCode != "0" {
		t.Errorf("exit_code = %v, want \"0\"", exitCode)
	}
}

func TestExecuteConsolidatedFirstCommandFails(t *testing.T) {
	os.Remove(RunnerResultFile)
	defer os.Remove(RunnerResultFile)

	var buf bytes.Buffer
	origStartRunner := startRunner
	startRunner = func() (io.WriteCloser, <-chan error, error) {
		jvmDone := fakeRunner(&buf, []int{1})
		return nopWriteCloser{&buf}, jvmDone, nil
	}
	defer func() { startRunner = origStartRunner }()

	commands := []ConsolidatedCommand{
		{Command: "update", Args: map[string]string{"PLUGIN_LIQUIBASE_URL": "jdbc:h2:mem:test"}},
		{Command: "tag", Args: map[string]string{"PLUGIN_LIQUIBASE_TAG": "v1.0"}},
	}

	pluginOutput := execution.NewOutput()
	err := executeConsolidated(Args{}, []string{}, commands, pluginOutput)
	if err != nil {
		t.Fatalf("executeConsolidated() error = %v, want nil", err)
	}

	exitCode := pluginOutput.GetProperty(OutputExitCode)
	if exitCode != "1" {
		t.Errorf("exit_code = %v, want \"1\"", exitCode)
	}
}

func TestExecuteConsolidatedCapturesStepOutput(t *testing.T) {
	os.Remove(RunnerResultFile)
	os.Remove(StepOutputFile)
	defer os.Remove(RunnerResultFile)
	defer os.Remove(StepOutputFile)

	var buf bytes.Buffer
	origStartRunner := startRunner
	startRunner = func() (io.WriteCloser, <-chan error, error) {
		jvmDone := make(chan error, 1)
		go func() {
			for {
				time.Sleep(50 * time.Millisecond)
				if buf.Len() == 0 {
					continue
				}
				buf.ReadString('\n')
				// Simulate Java writing both result and step output
				os.WriteFile(StepOutputFile, []byte(`{"flow":"test"}`), 0644)
				os.WriteFile(RunnerResultFile, []byte(`{"exitCode":0}`), 0644)
				return
			}
		}()
		return nopWriteCloser{&buf}, jvmDone, nil
	}
	defer func() { startRunner = origStartRunner }()

	commands := []ConsolidatedCommand{
		{Command: "update", Args: map[string]string{"PLUGIN_LIQUIBASE_URL": "jdbc:h2:mem:test"}},
	}

	pluginOutput := execution.NewOutput()
	args := Args{LiquibaseArgs: LiquibaseArgs{GenerateStepOutputs: "true"}}
	err := executeConsolidated(args, []string{}, commands, pluginOutput)
	if err != nil {
		t.Fatalf("executeConsolidated() error = %v", err)
	}

	stepOutput := pluginOutput.GetProperty(OutputStepOutput)
	if stepOutput != `{"flow":"test"}` {
		t.Errorf("step_output = %v, want %q", stepOutput, `{"flow":"test"}`)
	}
}
