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
	"github.com/sirupsen/logrus"
)

const (
	OutputExitCode   = "exit_code"
	OutputStepOutput = "step_output"
	StepOutputFile   = "/tmp/step_output.json"
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

	// Setup certificates
	certManager := NewCertManager(args.CertArgs)
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

	// Setup Google Cloud authentication
	gcpCleanup, err := setupGoogleCloudAuth(args.JSONKey)
	if err != nil {
		return fmt.Errorf("Google Cloud auth setup failed: %w", err)
	}
	if gcpCleanup != nil {
		cleanupFuncs = append(cleanupFuncs, gcpCleanup)
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

	// Load global options
	globalOptions, err := LoadGlobalOptions(args.GlobalOptionsFile)
	if err != nil {
		return fmt.Errorf("failed to load global options: %w", err)
	}
	logrus.Debugf("Loaded %d global options", len(globalOptions))

	// Build command arguments
	builder := NewCommandBuilder(globalOptions)
	commandArgs, err := builder.BuildArgs(args)
	if err != nil {
		return fmt.Errorf("failed to build command arguments: %w", err)
	}

	// Construct full command
	fullArgs := append([]string{args.LiquibaseBinary}, commandArgs...)
	logrus.Infof("Executing: %s", strings.Join(fullArgs, " "))

	// Remove step output file if it exists
	os.Remove(StepOutputFile)

	// Execute Liquibase
	exitCode, _, execErr := runCommandWithOutput(args.LiquibaseBinary, commandArgs...)
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
	if args.Command == "" {
		return fmt.Errorf("PLUGIN_COMMAND is required")
	}
	return nil
}
