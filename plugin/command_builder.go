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
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	pluginLiquibasePrefix = "PLUGIN_LIQUIBASE_"
)

// CommandBuilder builds the Liquibase CLI command arguments.
type CommandBuilder struct {
	globalOptions []string
	processedVars map[string]bool
}

// NewCommandBuilder creates a new CommandBuilder with the given global options.
func NewCommandBuilder(globalOptions []string) *CommandBuilder {
	return &CommandBuilder{
		globalOptions: globalOptions,
		processedVars: make(map[string]bool),
	}
}

// BuildArgs constructs the full argument list for the Liquibase command.
func (b *CommandBuilder) BuildArgs(args Args) ([]string, error) {
	var commandArgs []string

	// Process global options first (from the whitelist)
	globalArgs := b.processGlobalOptions()
	commandArgs = append(commandArgs, globalArgs...)

	// Add the command
	commandArgs = append(commandArgs, args.Command)

	// Process substitution properties
	if args.SubstituteLiquibase != "" {
		subArgs, err := b.processSubstitutionProperties(args.SubstituteLiquibase)
		if err != nil {
			return nil, fmt.Errorf("failed to process substitution properties: %w", err)
		}
		commandArgs = append(commandArgs, subArgs...)
	}

	// Process remaining PLUGIN_LIQUIBASE_* environment variables
	remainingArgs := b.processRemainingEnvVars()
	commandArgs = append(commandArgs, remainingArgs...)

	return commandArgs, nil
}

// processGlobalOptions processes environment variables for whitelisted global options.
func (b *CommandBuilder) processGlobalOptions() []string {
	var args []string

	for _, option := range b.globalOptions {
		envVar := OptionToEnvVar(option)
		value := os.Getenv(envVar)

		if value != "" {
			args = append(args, "--"+option, value)
			b.processedVars[envVar] = true
			// Unset the environment variable to hide sensitive values
			os.Unsetenv(envVar)
			logrus.Debugf("Added global option: --%s", option)
		}
	}

	return args
}

// processSubstitutionProperties decodes and processes the PLUGIN_SUBSTITUTE_LIQUIBASE value.
// The value is expected to be base64-encoded zstd-compressed JSON.
func (b *CommandBuilder) processSubstitutionProperties(encoded string) ([]string, error) {
	// Base64 decode
	compressed, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	// Zstd decompress
	decompressed, err := decompressZstd(compressed)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress zstd: %w", err)
	}

	if len(decompressed) == 0 {
		return nil, fmt.Errorf("decompressed data is empty")
	}

	// Parse JSON
	var properties map[string]interface{}
	if err := json.Unmarshal(decompressed, &properties); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Convert to -D arguments
	var args []string
	for key, value := range properties {
		arg := fmt.Sprintf("-D%s=%v", key, value)
		args = append(args, arg)
	}

	logrus.Debugf("Processed %d substitution properties", len(properties))
	return args, nil
}

// decompressZstd decompresses zstd-compressed data.
// Note: This uses zlib as a fallback since Go's standard library doesn't include zstd.
// The actual decompression is done by calling the zstd binary.
func decompressZstd(data []byte) ([]byte, error) {
	// Try zlib first (in case it's actually zlib compressed)
	zlibReader, err := zlib.NewReader(bytes.NewReader(data))
	if err == nil {
		defer zlibReader.Close()
		result, err := io.ReadAll(zlibReader)
		if err == nil {
			return result, nil
		}
	}

	// For zstd, we need to use the external binary
	// This is handled in the exec layer - return data as-is for now
	// The actual implementation will use os/exec to call zstd
	return decompressZstdExternal(data)
}

// decompressZstdExternal uses the zstd binary to decompress data.
func decompressZstdExternal(data []byte) ([]byte, error) {
	// Write to temp file
	tmpFile, err := os.CreateTemp("", "zstd-input-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write to temp file: %w", err)
	}
	tmpFile.Close()

	// Run zstd to decompress
	result, err := runCommand("zstd", "-d", "-c", tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("zstd decompression failed: %w", err)
	}

	return result, nil
}

// processRemainingEnvVars processes any PLUGIN_LIQUIBASE_* environment variables
// that weren't already processed as global options.
func (b *CommandBuilder) processRemainingEnvVars() []string {
	var args []string

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		name := parts[0]
		value := parts[1]

		if !strings.HasPrefix(name, pluginLiquibasePrefix) {
			continue
		}

		if b.processedVars[name] {
			continue
		}

		// Convert PLUGIN_LIQUIBASE_FOO_BAR to --foo-bar
		suffix := strings.TrimPrefix(name, pluginLiquibasePrefix)
		option := EnvVarToOption(suffix)

		args = append(args, "--"+option, value)
		logrus.Debugf("Added remaining option: --%s", option)
	}

	return args
}
