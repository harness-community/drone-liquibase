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
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/sirupsen/logrus"
)

// runCommand executes a command and returns its combined output.
func runCommand(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("command '%s %v' failed: %w\nstderr: %s", name, args, err, stderr.String())
	}

	// Combine stdout and stderr for commands that output to stderr (like java -version)
	output := stdout.Bytes()
	if len(output) == 0 {
		output = stderr.Bytes()
	}

	return output, nil
}

// runCommandWithOutput executes a command and streams output to stdout/stderr in real-time.
// Returns the exit code and captured output.
// This is a variable so that tests can replace it with a mock.
var runCommandWithOutput = runCommandWithOutputImpl

func runCommandWithOutputImpl(name string, args ...string) (int, []byte, error) {
	cmd := exec.Command(name, args...)

	// Capture output while streaming to stdout/stderr
	var outputBuf bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &outputBuf)
	cmd.Stderr = io.MultiWriter(os.Stderr, &outputBuf)

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return -1, outputBuf.Bytes(), err
		}
	}

	return exitCode, outputBuf.Bytes(), nil
}

// setupGoogleCloudAuth creates the service account key file for Google Cloud authentication.
func setupGoogleCloudAuth(jsonKey string) (cleanup func(), err error) {
	if jsonKey == "" {
		return nil, nil
	}

	serviceAccountKeyFile := "/tmp/harness-google-application-credentials.json"

	// Check if file already exists
	if fileExists(serviceAccountKeyFile) {
		logrus.Info("Service account key file already exists, skipping creation")
		return nil, nil
	}

	logrus.Info("Creating service account key file...")
	if err := os.WriteFile(serviceAccountKeyFile, []byte(jsonKey), 0600); err != nil {
		return nil, fmt.Errorf("failed to write service account key file: %w", err)
	}

	// Set environment variable
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", serviceAccountKeyFile)

	cleanup = func() {
		os.Remove(serviceAccountKeyFile)
	}

	return cleanup, nil
}
