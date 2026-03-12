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
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	maxRetries  = 3
	retryDelay  = 1 * time.Second
)

// LoadGlobalOptions reads the global options file with retry mechanism.
// Returns a slice of option names (in kebab-case).
func LoadGlobalOptions(filePath string) ([]string, error) {
	var options []string
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		options, lastErr = readOptionsFile(filePath)
		if lastErr == nil {
			return options, nil
		}

		logrus.Warnf("Failed to read global options file (attempt %d/%d): %v", attempt, maxRetries, lastErr)
		if attempt < maxRetries {
			time.Sleep(retryDelay)
		}
	}

	return nil, fmt.Errorf("failed to read '%s' after %d attempts: %w", filePath, maxRetries, lastErr)
}

func readOptionsFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var options []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			options = append(options, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(options) == 0 {
		logrus.Warn("No options were read from global options file")
	}

	return options, nil
}

// OptionToEnvVar converts a kebab-case option name to the corresponding
// environment variable name (e.g., "log-level" -> "PLUGIN_LIQUIBASE_LOG_LEVEL").
func OptionToEnvVar(option string) string {
	upper := strings.ToUpper(option)
	underscored := strings.ReplaceAll(upper, "-", "_")
	return "PLUGIN_LIQUIBASE_" + underscored
}

// EnvVarToOption converts an environment variable suffix to a kebab-case option name
// (e.g., "LOG_LEVEL" -> "log-level").
func EnvVarToOption(envVarSuffix string) string {
	lower := strings.ToLower(envVarSuffix)
	return strings.ReplaceAll(lower, "_", "-")
}
