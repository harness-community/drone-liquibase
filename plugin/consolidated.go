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
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/harness/liquibase-drone-plugin/internal/execution"
	"github.com/sirupsen/logrus"
)

const (
	// RunnerResultFile is where the Java CommandRunner writes its per-command exit code.
	RunnerResultFile = "/tmp/runner_result.json"

	runnerClasspath = "/liquibase/internal/lib/*:/liquibase/lib/*:/liquibase/internal/extensions/*"
)

// ConsolidatedCommand represents a single command in the consolidated execution flow.
type ConsolidatedCommand struct {
	Command string            `json:"command"`
	Args    map[string]string `json:"args"`
}

// RunnerCommand is sent to the Java CommandRunner via stdin as a JSON line.
type RunnerCommand struct {
	CliArgs []string          `json:"cliArgs"`
	EnvVars map[string]string `json:"envVars,omitempty"`
}

// RunnerResult is written by the Java CommandRunner to RunnerResultFile after each command.
type RunnerResult struct {
	ExitCode int `json:"exitCode"`
}

// decodeCommands decodes a base64-encoded JSON array of ConsolidatedCommand.
func decodeCommands(encoded string) ([]ConsolidatedCommand, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	var commands []ConsolidatedCommand
	if err := json.Unmarshal(decoded, &commands); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return commands, nil
}

// executeConsolidated runs all commands in a single JVM via the Java CommandRunner.
// Protocol: Go launches one JVM, sends commands as JSON lines on stdin,
// and reads per-command results from RunnerResultFile.
func executeConsolidated(args Args, globalOptions []string, commands []ConsolidatedCommand, pluginOutput *execution.Output) error {
	stdin, jvmDone, err := startRunner()
	if err != nil {
		return err
	}
	defer stdin.Close()

	for i, command := range commands {
		logrus.Info("")
		logrus.Info("========================================")
		logrus.Infof("Executing command %d of %d", i+1, len(commands))
		logrus.Info("========================================")
		logrus.Infof("Command: %s", command.Command)

		cliArgs, err := buildCommandArgs(args, globalOptions, command)
		if err != nil {
			pluginOutput.AddProperty(OutputExitCode, execution.OutputPropertyTypeSimple, "-1")
			return fmt.Errorf("failed to build args for command %d (%s): %w", i+1, command.Command, err)
		}

		envVars := command.Args

		exitCode, err := sendCommand(stdin, jvmDone, cliArgs, envVars)
		if err != nil {
			logrus.Errorf("Failed to execute command %d (%s): %v", i+1, command.Command, err)
			exitCode = -1
		}

		pluginOutput.AddProperty(OutputExitCode, execution.OutputPropertyTypeSimple, fmt.Sprintf("%d", exitCode))
		captureStepOutput(args, pluginOutput)

		if exitCode != 0 {
			return nil
		}

		logrus.Infof("Command '%s' completed successfully", command.Command)
	}

	logrus.Info("")
	logrus.Info("========================================")
	logrus.Infof("All %d commands completed successfully", len(commands))
	logrus.Info("========================================")
	return nil
}

// startRunner launches the Java CommandRunner JVM and returns a stdin writer
// and a channel that signals when the JVM exits.
// This is a variable so that tests can replace it with a mock.
var startRunner = startRunnerImpl

func startRunnerImpl() (io.WriteCloser, <-chan error, error) {
	// --add-opens flags allow CommandRunner to modify System.getenv() via reflection for per-command env vars
	javaArgs := []string{
		"--add-opens", "java.base/java.lang=ALL-UNNAMED",
		"--add-opens", "java.base/java.util=ALL-UNNAMED",
	}
	if javaOpts := os.Getenv("JAVA_OPTS"); javaOpts != "" {
		javaArgs = append(javaArgs, strings.Fields(javaOpts)...)
	}
	javaArgs = append(javaArgs, "-cp", runnerClasspath, CommandRunnerClass)

	cmd := exec.Command("java", javaArgs...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	var startErr error
	defer func() {
		if startErr != nil {
			stdin.Close()
		}
	}()

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		startErr = fmt.Errorf("failed to create stdout pipe: %w", err)
		return nil, nil, startErr
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		startErr = fmt.Errorf("failed to start command runner JVM: %w", err)
		return nil, nil, startErr
	}

	jvmDone := make(chan error, 1)
	go func() { jvmDone <- cmd.Wait() }()

	// Wait for the JVM to print "READY" on stdout, then forward all subsequent output to os.Stdout.
	ready := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Fprintln(os.Stdout, line)
			if line == "READY" {
				close(ready)
				break
			}
		}
		// Forward remaining stdout to os.Stdout
		io.Copy(os.Stdout, stdoutPipe)
	}()

	select {
	case <-ready:
		// JVM is alive and ready
	case err := <-jvmDone:
		if err != nil {
			startErr = fmt.Errorf("JVM exited before becoming ready: %w", err)
		} else {
			startErr = fmt.Errorf("JVM exited before printing READY")
		}
		return nil, nil, startErr
	case <-time.After(30 * time.Second):
		startErr = fmt.Errorf("timed out waiting for JVM READY signal")
		return nil, nil, startErr
	}

	return stdin, jvmDone, nil
}

// buildCommandArgs sets the per-command env vars, invokes CommandBuilder.BuildArgs,
// and cleans up env vars afterward. BuildArgs reads PLUGIN_LIQUIBASE_* from os.Getenv.
func buildCommandArgs(args Args, globalOptions []string, command ConsolidatedCommand) ([]string, error) {
	setKeys := setEnvVars(command.Args)
	defer unsetEnvVars(setKeys)

	execArgs := args
	execArgs.Command = command.Command
	if v, ok := command.Args["PLUGIN_SUBSTITUTE_LIQUIBASE"]; ok {
		execArgs.SubstituteLiquibase = v
	}
	if v, ok := command.Args["GENERATE_STEP_OUTPUTS"]; ok {
		execArgs.GenerateStepOutputs = v
	}

	return NewCommandBuilder(globalOptions).BuildArgs(execArgs)
}

// sendCommand sends one command to the runner via stdin, waits for the result file,
// and returns the exit code.
func sendCommand(stdin io.Writer, jvmDone <-chan error, cliArgs []string, envVars map[string]string) (int, error) {
	os.Remove(RunnerResultFile)
	os.Remove(StepOutputFile)

	runnerCmd := RunnerCommand{CliArgs: cliArgs, EnvVars: envVars}
	cmdJSON, err := json.Marshal(runnerCmd)
	if err != nil {
		return -1, fmt.Errorf("failed to encode command: %w", err)
	}

	if _, err := fmt.Fprintf(stdin, "%s\n", cmdJSON); err != nil {
		return -1, fmt.Errorf("failed to send command to runner: %w", err)
	}

	result, err := waitForRunnerResult(jvmDone)
	if err != nil {
		return -1, err
	}
	return result.ExitCode, nil
}

// setEnvVars sets the given key-value pairs in the process environment
// and returns the keys that were set (for cleanup).
func setEnvVars(vars map[string]string) []string {
	keys := make([]string, 0, len(vars))
	for k, v := range vars {
		os.Setenv(k, v)
		keys = append(keys, k)
	}
	return keys
}

// unsetEnvVars removes the given keys from the process environment.
func unsetEnvVars(keys []string) {
	for _, k := range keys {
		os.Unsetenv(k)
	}
}

// waitForRunnerResult polls for the runner result file written by CommandRunner.
// Runs indefinitely until the result file appears or the JVM exits.
func waitForRunnerResult(jvmDone <-chan error) (*RunnerResult, error) {
	for {
		// Check for the result file first — the JVM may write the file and exit
		// almost simultaneously, so we must read the file before consuming from
		// jvmDone to avoid missing a valid result.
		if fileExists(RunnerResultFile) {
			data, err := os.ReadFile(RunnerResultFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read runner result: %w", err)
			}
			os.Remove(RunnerResultFile)

			var result RunnerResult
			if err := json.Unmarshal(data, &result); err != nil {
				return nil, fmt.Errorf("failed to parse runner result: %w", err)
			}
			return &result, nil
		}

		select {
		case err := <-jvmDone:
			if err != nil {
				return nil, fmt.Errorf("JVM process crashed: %w", err)
			}
			return nil, fmt.Errorf("JVM process exited unexpectedly")
		default:
		}

		time.Sleep(500 * time.Millisecond)
	}
}
