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

// populateAuthArgs overlays auth-related values from the first command's args
// map onto the Args struct so that certificate, Kerberos, and GCP auth setup
// in Exec() works correctly for the consolidated execution flow.
func populateAuthArgs(args *Args, cmdArgs map[string]string) {
	// Kerberos args
	if v, ok := cmdArgs["PLUGIN_KERBEROS_USER_PRINCIPAL"]; ok {
		args.KerberosArgs.UserPrincipal = v
	}
	if v, ok := cmdArgs["PLUGIN_KERBEROS_PASSWORD"]; ok {
		args.KerberosArgs.Password = v
	}
	if v, ok := cmdArgs["PLUGIN_KERBEROS_KEYTAB_FILE_PATH"]; ok {
		args.KerberosArgs.KeytabFilePath = v
	}

	// GCP auth
	if v, ok := cmdArgs["PLUGIN_JSON_KEY"]; ok {
		args.LiquibaseArgs.JSONKey = v
	}

	// Cert args (only override if present, keep defaults otherwise)
	if v, ok := cmdArgs["PLUGIN_CERTS_DIR"]; ok {
		args.CertArgs.CertsDir = v
	}
	if v, ok := cmdArgs["PLUGIN_SSL_CA_CERT_PATH"]; ok {
		args.CertArgs.SSLCACertPath = v
	}
	if v, ok := cmdArgs["PLUGIN_CLIENT_CERT_PATH"]; ok {
		args.CertArgs.ClientCertPath = v
	}
	if v, ok := cmdArgs["PLUGIN_CLIENT_KEY_PATH"]; ok {
		args.CertArgs.ClientKeyPath = v
	}
	if v, ok := cmdArgs["PLUGIN_STORE_PASSWORD"]; ok {
		args.CertArgs.StorePassword = v
	}
}

// executeConsolidated runs multiple Liquibase commands sequentially.
func executeConsolidated(args Args, globalOptions []string, commands []ConsolidatedCommand, pluginOutput *execution.Output) error {
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

		fullArgs := append([]string{args.LiquibaseBinary}, commandArgs...)
		logrus.Info("")
		logrus.Info("----------------------------------------")
		logrus.Info("Command Arguments")
		logrus.Info("----------------------------------------")
		logrus.Infof("%s", strings.Join(fullArgs, " "))
		logrus.Info("----------------------------------------")
		logrus.Info("")

		// Remove step output file before each command
		os.Remove(StepOutputFile)

		exitCode, _, execErr := runCommandWithOutput(args.LiquibaseBinary, commandArgs...)
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
