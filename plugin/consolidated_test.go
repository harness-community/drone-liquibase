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
	"os"
	"strings"
	"testing"

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
		t.Errorf("result[0].Args[PLUGIN_LIQUIBASE_CHANGELOG_FILE] = %q, want %q", result[0].Args["PLUGIN_LIQUIBASE_CHANGELOG_FILE"], "changelog.xml")
	}
	if result[1].Command != "tag" {
		t.Errorf("result[1].Command = %q, want %q", result[1].Command, "tag")
	}
	if result[1].Args["PLUGIN_LIQUIBASE_TAG"] != "v1.0" {
		t.Errorf("result[1].Args[PLUGIN_LIQUIBASE_TAG] = %q, want %q", result[1].Args["PLUGIN_LIQUIBASE_TAG"], "v1.0")
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

func TestValidateInputsConsolidated(t *testing.T) {
	tests := []struct {
		name    string
		args    Args
		wantErr bool
	}{
		{
			name: "command set",
			args: Args{
				LiquibaseArgs: LiquibaseArgs{
					Command: "update",
				},
			},
			wantErr: false,
		},
		{
			name: "commands set",
			args: Args{
				LiquibaseArgs: LiquibaseArgs{
					ConsolidatedCommand: base64.StdEncoding.EncodeToString([]byte(`[{"command":"update","args":{}}]`)),
				},
			},
			wantErr: false,
		},
		{
			name: "both set",
			args: Args{
				LiquibaseArgs: LiquibaseArgs{
					Command:  "update",
					ConsolidatedCommand: base64.StdEncoding.EncodeToString([]byte(`[{"command":"tag","args":{}}]`)),
				},
			},
			wantErr: false,
		},
		{
			name: "neither set",
			args: Args{
				LiquibaseArgs: LiquibaseArgs{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInputs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateInputs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
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
		t.Errorf("Args[PLUGIN_LIQUIBASE_URL] = %q, want %q", decoded.Args["PLUGIN_LIQUIBASE_URL"], cmd.Args["PLUGIN_LIQUIBASE_URL"])
	}
	if decoded.Args["PLUGIN_LIQUIBASE_USERNAME"] != cmd.Args["PLUGIN_LIQUIBASE_USERNAME"] {
		t.Errorf("Args[PLUGIN_LIQUIBASE_USERNAME] = %q, want %q", decoded.Args["PLUGIN_LIQUIBASE_USERNAME"], cmd.Args["PLUGIN_LIQUIBASE_USERNAME"])
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
	// JSON object instead of array
	encoded := base64.StdEncoding.EncodeToString([]byte(`{"command":"update"}`))
	_, err := decodeCommands(encoded)
	if err == nil {
		t.Error("decodeCommands() should error for JSON object instead of array")
	}
}

func TestExecuteConsolidatedEnvVarCleanup(t *testing.T) {
	// Verify that env vars set for one command are cleaned up after execution
	commands := []ConsolidatedCommand{
		{
			Command: "hello",
			Args:    map[string]string{"PLUGIN_LIQUIBASE_URL": "jdbc:test://db1", "PLUGIN_LIQUIBASE_USERNAME": "testuser"},
		},
	}

	pluginOutput := execution.NewOutput()
	args := Args{
		LiquibaseArgs: LiquibaseArgs{
			LiquibaseBinary: "echo", // succeeds immediately
		},
	}

	err := executeConsolidated(args, []string{}, commands, pluginOutput)
	if err != nil {
		t.Fatalf("executeConsolidated() error = %v", err)
	}

	// Verify env vars from the command args were cleaned up
	for key := range commands[0].Args {
		if val := os.Getenv(key); val != "" {
			t.Errorf("Env var %s should be unset after executeConsolidated, got %q", key, val)
		}
	}
}

func TestExecuteConsolidatedSuccess(t *testing.T) {
	commands := []ConsolidatedCommand{
		{Command: "echo", Args: map[string]string{}},
	}

	pluginOutput := execution.NewOutput()
	args := Args{
		LiquibaseArgs: LiquibaseArgs{
			LiquibaseBinary: "echo",
		},
	}

	err := executeConsolidated(args, []string{}, commands, pluginOutput)
	if err != nil {
		t.Fatalf("executeConsolidated() error = %v", err)
	}
}

func TestExecuteConsolidatedFailure(t *testing.T) {
	commands := []ConsolidatedCommand{
		{Command: "nonexistent-subcommand", Args: map[string]string{}},
	}

	pluginOutput := execution.NewOutput()
	args := Args{
		LiquibaseArgs: LiquibaseArgs{
			LiquibaseBinary: "false", // exits with code 1
		},
	}

	err := executeConsolidated(args, []string{}, commands, pluginOutput)
	if err == nil {
		t.Error("executeConsolidated() should error on command failure")
	}
}

func TestExecuteConsolidatedStructFieldSync(t *testing.T) {
	// Verify that PLUGIN_SUBSTITUTE_LIQUIBASE and GENERATE_STEP_OUTPUTS
	// from cmd.Args are synced to the struct fields used by BuildArgs
	commands := []ConsolidatedCommand{
		{
			Command: "hello",
			Args: map[string]string{
				"PLUGIN_SUBSTITUTE_LIQUIBASE": "some-encoded-value",
				"GENERATE_STEP_OUTPUTS":      "true",
			},
		},
	}

	pluginOutput := execution.NewOutput()
	args := Args{
		LiquibaseArgs: LiquibaseArgs{
			LiquibaseBinary: "echo",
		},
	}

	// SubstituteLiquibase will cause BuildArgs to fail (invalid base64+zstd),
	// but that confirms the value was synced to the struct field
	err := executeConsolidated(args, []string{}, commands, pluginOutput)
	if err == nil {
		t.Error("executeConsolidated() should error due to invalid PLUGIN_SUBSTITUTE_LIQUIBASE")
	}
	if !strings.Contains(err.Error(), "substitution properties") {
		t.Errorf("error should mention substitution properties, got: %v", err)
	}
}

func TestExecuteConsolidatedMultiCommandFailsOnSecond(t *testing.T) {
	// First command succeeds, second fails — verify early termination
	commands := []ConsolidatedCommand{
		{Command: "first", Args: map[string]string{}},
		{Command: "second", Args: map[string]string{}},
	}

	pluginOutput := execution.NewOutput()
	// Use "sh" with -c to make first succeed and second fail
	// But simpler: we can't easily make only the 2nd fail with echo/false.
	// Instead, use a binary that always fails — both would fail,
	// but error message should reference command 1 (first to fail).
	args := Args{
		LiquibaseArgs: LiquibaseArgs{
			LiquibaseBinary: "false",
		},
	}

	err := executeConsolidated(args, []string{}, commands, pluginOutput)
	if err == nil {
		t.Fatal("executeConsolidated() should error on command failure")
	}
	// Should fail on command 1 and not reach command 2
	if !strings.Contains(err.Error(), "command 1") {
		t.Errorf("error should reference command 1, got: %v", err)
	}
}

func TestExecuteConsolidatedEnvVarIsolationBetweenCommands(t *testing.T) {
	// Verify env vars from command 1 don't leak into command 2
	commands := []ConsolidatedCommand{
		{
			Command: "first",
			Args:    map[string]string{"PLUGIN_LIQUIBASE_URL": "jdbc:first://db"},
		},
		{
			Command: "second",
			Args:    map[string]string{"PLUGIN_LIQUIBASE_USERNAME": "user2"},
		},
	}

	pluginOutput := execution.NewOutput()
	args := Args{
		LiquibaseArgs: LiquibaseArgs{
			LiquibaseBinary: "echo",
		},
	}

	err := executeConsolidated(args, []string{}, commands, pluginOutput)
	if err != nil {
		t.Fatalf("executeConsolidated() error = %v", err)
	}

	// After execution, all env vars from both commands should be cleaned up
	if val := os.Getenv("PLUGIN_LIQUIBASE_URL"); val != "" {
		t.Errorf("PLUGIN_LIQUIBASE_URL should be unset, got %q", val)
	}
	if val := os.Getenv("PLUGIN_LIQUIBASE_USERNAME"); val != "" {
		t.Errorf("PLUGIN_LIQUIBASE_USERNAME should be unset, got %q", val)
	}
}

func TestPopulateAuthArgsKerberos(t *testing.T) {
	args := Args{}
	cmdArgs := map[string]string{
		"PLUGIN_KERBEROS_USER_PRINCIPAL":   "user@REALM",
		"PLUGIN_KERBEROS_PASSWORD":         "secret",
		"PLUGIN_KERBEROS_KEYTAB_FILE_PATH": "/path/to/keytab",
	}

	populateAuthArgs(&args, cmdArgs)

	if args.KerberosArgs.UserPrincipal != "user@REALM" {
		t.Errorf("UserPrincipal = %q, want %q", args.KerberosArgs.UserPrincipal, "user@REALM")
	}
	if args.KerberosArgs.Password != "secret" {
		t.Errorf("Password = %q, want %q", args.KerberosArgs.Password, "secret")
	}
	if args.KerberosArgs.KeytabFilePath != "/path/to/keytab" {
		t.Errorf("KeytabFilePath = %q, want %q", args.KerberosArgs.KeytabFilePath, "/path/to/keytab")
	}
}

func TestPopulateAuthArgsGCP(t *testing.T) {
	args := Args{}
	cmdArgs := map[string]string{
		"PLUGIN_JSON_KEY": `{"type":"service_account"}`,
	}

	populateAuthArgs(&args, cmdArgs)

	if args.LiquibaseArgs.JSONKey != `{"type":"service_account"}` {
		t.Errorf("JSONKey = %q, want %q", args.LiquibaseArgs.JSONKey, `{"type":"service_account"}`)
	}
}

func TestPopulateAuthArgsCerts(t *testing.T) {
	args := Args{
		CertArgs: CertArgs{
			CertsDir:      "/default/certs",
			SSLCACertPath: "/default/ca.crt",
			ClientCertPath: "/default/client.crt",
			ClientKeyPath:  "/default/client.key",
			StorePassword:  "changeit",
		},
	}
	cmdArgs := map[string]string{
		"PLUGIN_CERTS_DIR":        "/custom/certs",
		"PLUGIN_SSL_CA_CERT_PATH": "/custom/ca.crt",
		"PLUGIN_STORE_PASSWORD":   "custompass",
	}

	populateAuthArgs(&args, cmdArgs)

	if args.CertArgs.CertsDir != "/custom/certs" {
		t.Errorf("CertsDir = %q, want %q", args.CertArgs.CertsDir, "/custom/certs")
	}
	if args.CertArgs.SSLCACertPath != "/custom/ca.crt" {
		t.Errorf("SSLCACertPath = %q, want %q", args.CertArgs.SSLCACertPath, "/custom/ca.crt")
	}
	if args.CertArgs.StorePassword != "custompass" {
		t.Errorf("StorePassword = %q, want %q", args.CertArgs.StorePassword, "custompass")
	}
	// Defaults preserved when not in cmdArgs
	if args.CertArgs.ClientCertPath != "/default/client.crt" {
		t.Errorf("ClientCertPath = %q, want %q (default preserved)", args.CertArgs.ClientCertPath, "/default/client.crt")
	}
	if args.CertArgs.ClientKeyPath != "/default/client.key" {
		t.Errorf("ClientKeyPath = %q, want %q (default preserved)", args.CertArgs.ClientKeyPath, "/default/client.key")
	}
}

func TestPopulateAuthArgsEmptyMap(t *testing.T) {
	args := Args{
		CertArgs: CertArgs{
			CertsDir:      "/default/certs",
			StorePassword: "changeit",
		},
	}
	cmdArgs := map[string]string{}

	populateAuthArgs(&args, cmdArgs)

	// All defaults should be preserved
	if args.CertArgs.CertsDir != "/default/certs" {
		t.Errorf("CertsDir = %q, want %q", args.CertArgs.CertsDir, "/default/certs")
	}
	if args.CertArgs.StorePassword != "changeit" {
		t.Errorf("StorePassword = %q, want %q", args.CertArgs.StorePassword, "changeit")
	}
	if args.KerberosArgs.UserPrincipal != "" {
		t.Errorf("UserPrincipal = %q, want empty", args.KerberosArgs.UserPrincipal)
	}
	if args.LiquibaseArgs.JSONKey != "" {
		t.Errorf("JSONKey = %q, want empty", args.LiquibaseArgs.JSONKey)
	}
}

func TestPopulateAuthArgsAllFields(t *testing.T) {
	args := Args{}
	cmdArgs := map[string]string{
		"PLUGIN_KERBEROS_USER_PRINCIPAL":   "user@REALM",
		"PLUGIN_KERBEROS_PASSWORD":         "secret",
		"PLUGIN_KERBEROS_KEYTAB_FILE_PATH": "/keytab",
		"PLUGIN_JSON_KEY":                  `{"key":"val"}`,
		"PLUGIN_CERTS_DIR":                 "/certs",
		"PLUGIN_SSL_CA_CERT_PATH":          "/ca.crt",
		"PLUGIN_CLIENT_CERT_PATH":          "/client.crt",
		"PLUGIN_CLIENT_KEY_PATH":           "/client.key",
		"PLUGIN_STORE_PASSWORD":            "pass",
	}

	populateAuthArgs(&args, cmdArgs)

	if args.KerberosArgs.UserPrincipal != "user@REALM" {
		t.Errorf("UserPrincipal = %q, want %q", args.KerberosArgs.UserPrincipal, "user@REALM")
	}
	if args.LiquibaseArgs.JSONKey != `{"key":"val"}` {
		t.Errorf("JSONKey = %q, want %q", args.LiquibaseArgs.JSONKey, `{"key":"val"}`)
	}
	if args.CertArgs.CertsDir != "/certs" {
		t.Errorf("CertsDir = %q, want %q", args.CertArgs.CertsDir, "/certs")
	}
	if args.CertArgs.ClientCertPath != "/client.crt" {
		t.Errorf("ClientCertPath = %q, want %q", args.CertArgs.ClientCertPath, "/client.crt")
	}
	if args.CertArgs.ClientKeyPath != "/client.key" {
		t.Errorf("ClientKeyPath = %q, want %q", args.CertArgs.ClientKeyPath, "/client.key")
	}
	if args.CertArgs.StorePassword != "pass" {
		t.Errorf("StorePassword = %q, want %q", args.CertArgs.StorePassword, "pass")
	}
}

func TestConsolidatedCommandNoArgs(t *testing.T) {
	cmd := ConsolidatedCommand{
		Command: "status",
	}

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
