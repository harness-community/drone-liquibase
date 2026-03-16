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

package main

import (
	"context"
	"os/exec"

	"github.com/harness/liquibase-drone-plugin/internal/logger"
	"github.com/harness/liquibase-drone-plugin/plugin"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
)

// requiredCommands lists commands that must be available for the plugin to function.
// These match the bash script's dependency check: jq, zstd, base64, plus keytool and openssl for certs.
var requiredCommands = []string{"zstd", "keytool", "openssl"}

// optionalCommands lists commands that are only needed for specific features.
var optionalCommands = []string{"kinit"} // Only needed for Kerberos auth

func main() {
	logrus.SetFormatter(logger.GetDefaultLoggerFormatterWithoutTimestamp())

	// Validate required dependencies early (matching bash: for cmd in jq zstd base64; do ...)
	if err := validateDependencies(); err != nil {
		logrus.Fatalln(err)
	}

	var args plugin.Args
	if err := envconfig.Process("", &args); err != nil {
		logrus.Fatalln(err)
	}

	level, err := logrus.ParseLevel(args.LogLevel)
	if err != nil {
		logrus.SetLevel(logrus.InfoLevel)
		logrus.Warnf("Invalid log level '%s', defaulting to info", args.LogLevel)
	} else {
		logrus.SetLevel(level)
	}

	if err := plugin.Exec(context.Background(), args); err != nil {
		logrus.Fatalln(err)
	}
}

// validateDependencies checks that required commands are available in PATH.
func validateDependencies() error {
	for _, cmd := range requiredCommands {
		if _, err := exec.LookPath(cmd); err != nil {
			logrus.Errorf("Missing required command: %s", cmd)
			return err
		}
	}

	// Log warnings for optional commands
	for _, cmd := range optionalCommands {
		if _, err := exec.LookPath(cmd); err != nil {
			logrus.Debugf("Optional command not found: %s (only needed for Kerberos)", cmd)
		}
	}

	return nil
}
