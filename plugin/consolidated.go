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

	"github.com/harness/liquibase-drone-plugin/internal/execution"
	"github.com/sirupsen/logrus"
)

// ConsolidatedCommand represents a single command in the consolidated execution flow.
type ConsolidatedCommand struct {
	Command string            `json:"command"`
	Args    map[string]string `json:"args"`
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

// executeConsolidated runs multiple Liquibase commands sequentially.
// authOverrides are env var overrides from auth setup (e.g. GCP OIDC) that
// take precedence over per-command args to prevent commands from clobbering
// auth-configured values.
func executeConsolidated(args Args, globalOptions []string, commands []ConsolidatedCommand, authOverrides map[string]string, pluginOutput *execution.Output) error {
	logrus.Info("========================================")
	logrus.Info("Running consolidated execution flow...")
	logrus.Info("========================================")
	logrus.Infof("Found %d commands to execute", len(commands))

	for i, cmd := range commands {
		logrus.Info("")
		logrus.Info("========================================")
		logrus.Infof("Executing command %d of %d", i+1, len(commands))
		logrus.Info("========================================")
		logrus.Infof("Command: %s", cmd.Command)

		// Set env vars from command args (keys are already full env var names,
		// e.g. PLUGIN_LIQUIBASE_USERNAME, PLUGIN_BEARER_TOKEN)
		var setEnvVars []string
		for key, value := range cmd.Args {
			os.Setenv(key, value)
			setEnvVars = append(setEnvVars, key)
		}

		// Apply auth overrides on top — auth-configured values always win
		for key, value := range authOverrides {
			os.Setenv(key, value)
			setEnvVars = append(setEnvVars, key)
		}

		// Build per-command args, syncing struct fields that BuildArgs reads
		// directly from the struct rather than from env vars
		execArgs := args
		execArgs.Command = cmd.Command
		if v, ok := cmd.Args["PLUGIN_SUBSTITUTE_LIQUIBASE"]; ok {
			execArgs.SubstituteLiquibase = v
		}
		if v, ok := cmd.Args["GENERATE_STEP_OUTPUTS"]; ok {
			execArgs.GenerateStepOutputs = v
		}

		builder := NewCommandBuilder(globalOptions)
		commandArgs, err := builder.BuildArgs(execArgs)
		if err != nil {
			pluginOutput.AddProperty(OutputExitCode, execution.OutputPropertyTypeSimple, fmt.Sprintf("%d", -1))
			return fmt.Errorf("failed to build args for command %d (%s): %w", i+1, cmd.Command, err)
		}

		fullArgs := append([]string{LiquibaseBinary}, commandArgs...)
		logrus.Info("")
		logrus.Info("----------------------------------------")
		logrus.Info("Command Arguments")
		logrus.Info("----------------------------------------")
		logrus.Infof("%s", strings.Join(fullArgs, " "))
		logrus.Info("----------------------------------------")
		logrus.Info("")

		// Remove step output file before each command
		os.Remove(StepOutputFile)

		exitCode, _, execErr := runCommandWithOutput(LiquibaseBinary, commandArgs...)
		if execErr != nil {
			logrus.Errorf("Failed to execute command %d (%s): %v", i+1, cmd.Command, execErr)
			exitCode = -1
		}

		// Clean up env vars set for this command
		for _, envVar := range setEnvVars {
			os.Unsetenv(envVar)
		}

		if exitCode != 0 {
			pluginOutput.AddProperty(OutputExitCode, execution.OutputPropertyTypeSimple, fmt.Sprintf("%d", exitCode))
			return fmt.Errorf("command %d (%s) failed with exit code %d", i+1, cmd.Command, exitCode)
		}

		logrus.Infof("Command '%s' completed successfully", cmd.Command)
	}

	logrus.Info("")
	logrus.Info("========================================")
	logrus.Infof("All %d commands completed successfully", len(commands))
	logrus.Info("========================================")

	// All commands succeeded
	pluginOutput.AddProperty(OutputExitCode, execution.OutputPropertyTypeSimple, "0")

	// Write step output if enabled
	if args.GenerateStepOutputs == "true" {
		if fileExists(StepOutputFile) {
			stepOutput, err := os.ReadFile(StepOutputFile)
			if err != nil {
				logrus.Warnf("Failed to read step output file: %v", err)
			} else {
				pluginOutput.AddProperty(OutputStepOutput, execution.OutputPropertyTypeSimple, strings.TrimRight(string(stepOutput), "\n"))
			}
			os.Remove(StepOutputFile)
		}
	}

	return nil
}
