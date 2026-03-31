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
	"fmt"
	"os"
	"strings"

	"github.com/harness/liquibase-drone-plugin/internal/execution"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
)

const (
	OutputExitCode   = "exit_code"
	OutputStepOutput = "step_output"
	StepOutputFile   = "/tmp/step_output.json"

	LicenseGlobPattern = "/tmp/cert/*.jar.b64"
	LicenseTargetDir   = "/liquibase/lib"
)

// Exec executes the Liquibase plugin.
func Exec(args Args) (mainErr error) {
	logrus.Info("Starting Liquibase Plugin")

	pluginOutput := execution.NewOutput()

	// Cleanup functions to run on exit
	var cleanupFuncs []func()
	defer func() {
		for _, cleanup := range cleanupFuncs {
			if cleanup != nil {
				cleanup()
			}
		}
	}()

	defer func() {
		if r := recover(); r != nil {
			mainErr = fmt.Errorf("panic: %v", r)
		}

		if mainErr == nil {
			logrus.Info("Liquibase plugin successfully completed")
			pluginOutput.SetExecutionStatus(execution.ExecutionStatusSuccess)
		} else {
			logrus.Errorf("Liquibase plugin execution failed: %v", mainErr)
			pluginOutput.SetExecutionStatus(execution.ExecutionStatusFailure)
			errorResponse := &execution.Response{}
			_ = execution.HandleError(mainErr, errorResponse)
			pluginOutput.SetExecutionResponse(*errorResponse)
			mainErr = fmt.Errorf("%s", execution.ErrorToString(mainErr))
		}

		_, e := pluginOutput.CreateOutputFile(args.DroneOutputFile)
		if e != nil {
			logrus.Errorf("Failed to create drone output file: %v", e)
		}
	}()

	// Validate required inputs
	if err := validateInputs(args); err != nil {
		return err
	}

	// Detect and export JAVA_HOME
	javaHome := detectJavaHome()
	if javaHome == "" {
		logrus.Warn("JAVA_HOME not detected, some features may not work")
	}

	// Load global options
	globalOptions, err := LoadGlobalOptions(GlobalOptionsFile)
	if err != nil {
		return fmt.Errorf("failed to load global options: %w", err)
	}
	logrus.Debugf("Loaded %d global options", len(globalOptions))

	// For consolidated flow, decode commands early and export auth args
	// from the first command as env vars so that cert, Kerberos, and GCP
	// auth setup works correctly. Re-process the struct to pick them up.

	var commands []ConsolidatedCommand
	if args.ConsolidatedCommand != "" {
		commands, err = decodeCommands(args.ConsolidatedCommand)
		if err != nil {
			return fmt.Errorf("failed to decode PLUGIN_COMMANDS: %w", err)
		}
		if len(commands) > 0 {
			for key, value := range commands[0].Args {
				os.Setenv(key, value)
			}
			if err := envconfig.Process("", &args); err != nil {
				return fmt.Errorf("failed to re-process args after exporting auth env vars: %w", err)
			}
		}
	}

	// Setup certificates
	certManager := NewCertManager(args.CertArgs, javaHome)
	javaOptsFromCerts, err := certManager.SetupCertificates(args.CertArgs)
	if err != nil {
		logrus.Warnf("Certificate setup warning: %v", err)
	}

	// Setup Kerberos authentication
	kerberosManager := NewKerberosManager()
	javaOptsFromKerberos, kerberosCleanup, err := kerberosManager.Authenticate(args.KerberosArgs)
	if err != nil {
		return fmt.Errorf("Kerberos authentication failed: %w", err)
	}
	if kerberosCleanup != nil {
		cleanupFuncs = append(cleanupFuncs, kerberosCleanup)
	}

	// Auto-discover and install any base64-encoded license JARs
	discoverAndInstallLicenseFiles(LicenseGlobPattern, LicenseTargetDir)

	// Setup Google Cloud authentication
	gcpCleanup, err := setupGoogleCloudAuth(args.JSONKey)
	if err != nil {
		return fmt.Errorf("Google Cloud auth setup failed: %w", err)
	}
	if gcpCleanup != nil {
		cleanupFuncs = append(cleanupFuncs, gcpCleanup)
	}

	envOverrides := make(map[string]string)
	if err := ModifyGcpOidcAuthOverrides(args.GCPOIDCArgs, os.Getenv(envPluginLiquibaseURL), envOverrides); err != nil {
		return fmt.Errorf("GCP OIDC auth setup failed: %w", err)
	}

	// Set JAVA_OPTS
	var javaOptsParts []string
	if existingOpts := os.Getenv("JAVA_OPTS"); existingOpts != "" {
		javaOptsParts = append(javaOptsParts, existingOpts)
	}
	if javaOptsFromCerts != "" {
		javaOptsParts = append(javaOptsParts, javaOptsFromCerts)
	}
	if javaOptsFromKerberos != "" {
		javaOptsParts = append(javaOptsParts, javaOptsFromKerberos)
	}
	if len(javaOptsParts) > 0 {
		os.Setenv("JAVA_OPTS", strings.Join(javaOptsParts, " "))
	}

	// Consolidated execution flow: multiple commands via PLUGIN_COMMANDS
	if args.ConsolidatedCommand != "" {
		return executeConsolidated(args, globalOptions, commands, envOverrides, pluginOutput)
	}

	// Single command execution flow — apply auth overrides to env vars
	applyEnvOverrides(envOverrides)
	// Cleanup: unset env overrides on exit
	if len(envOverrides) > 0 {
		cleanupFuncs = append(cleanupFuncs, func() {
			unsetEnvOverrides(envOverrides)
		})
	}
	builder := NewCommandBuilder(globalOptions)
	commandArgs, err := builder.BuildArgs(args)
	if err != nil {
		return fmt.Errorf("failed to build command arguments: %w", err)
	}

	// Construct full command
	fullArgs := append([]string{LiquibaseBinary}, commandArgs...)
	logrus.Infof("Executing: %s", strings.Join(fullArgs, " "))

	// Remove step output file if it exists
	os.Remove(StepOutputFile)

	// Execute Liquibase
	exitCode, _, execErr := runCommandWithOutput(LiquibaseBinary, commandArgs...)
	if execErr != nil {
		logrus.Errorf("Failed to execute Liquibase: %v", execErr)
		exitCode = -1
	}

	// Write exit code to DRONE_OUTPUT
	pluginOutput.AddProperty(OutputExitCode, execution.OutputPropertyTypeSimple, fmt.Sprintf("%d", exitCode))

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

// validateInputs validates the required plugin inputs.
func validateInputs(args Args) error {
	if args.Command == "" && args.ConsolidatedCommand == "" {
		return fmt.Errorf("PLUGIN_COMMAND or PLUGIN_COMMANDS is required")
	}
	return nil
}
