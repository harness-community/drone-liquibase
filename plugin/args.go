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

import "time"

// Args provides plugin execution arguments.
type Args struct {
	Pipeline
	LiquibaseArgs
	CertArgs
	KerberosArgs

	LogLevel        string        `envconfig:"PLUGIN_LOG_LEVEL" default:"info"`
	Timeout         time.Duration `envconfig:"PLUGIN_TIMEOUT" default:"30m"`
	DroneOutputFile string        `envconfig:"DRONE_OUTPUT" default:"/tmp/output.out"`
}

// LiquibaseArgs contains arguments specific to Liquibase execution.
type LiquibaseArgs struct {
	// Command is the Liquibase command to execute (e.g., update, rollback)
	Command string `envconfig:"PLUGIN_COMMAND"`

	// ConsolidatedCommand is a base64-encoded JSON array of commands for consolidated execution
	ConsolidatedCommand string `envconfig:"PLUGIN_COMMANDS"`

	// SubstituteLiquibase contains base64+zstd encoded JSON for changelog substitutions
	SubstituteLiquibase string `envconfig:"PLUGIN_SUBSTITUTE_LIQUIBASE"`

	// JSONKey contains Google Cloud service account key for Spanner
	JSONKey string `envconfig:"PLUGIN_JSON_KEY"`

	// GenerateStepOutputs enables step output generation
	GenerateStepOutputs string `envconfig:"GENERATE_STEP_OUTPUTS"`

	// GlobalOptionsFile is the path to the global options file
	GlobalOptionsFile string `envconfig:"PLUGIN_GLOBAL_OPTIONS_FILE" default:"/resources/global_options.txt"`

	// LiquibaseBinary is the path to the liquibase binary
	LiquibaseBinary string `envconfig:"PLUGIN_LIQUIBASE_BINARY" default:"/liquibase/liquibase"`
}

// CertArgs contains certificate-related configuration.
type CertArgs struct {
	// CertsDir is the user-writable directory for certificates
	CertsDir string `envconfig:"PLUGIN_CERTS_DIR" default:"/harness/certs"`

	// SSLCACertPath is the path to the root CA certificate
	SSLCACertPath string `envconfig:"PLUGIN_SSL_CA_CERT_PATH" default:"/etc/ssl/certs/dbops/root_ca.crt"`

	// ClientCertPath is the path to the client certificate
	ClientCertPath string `envconfig:"PLUGIN_CLIENT_CERT_PATH" default:"/etc/ssl/certs/dbops/client.crt"`

	// ClientKeyPath is the path to the client private key
	ClientKeyPath string `envconfig:"PLUGIN_CLIENT_KEY_PATH" default:"/etc/ssl/certs/dbops/client.key"`

	// StorePassword is the password for the keystore/truststore
	StorePassword string `envconfig:"PLUGIN_STORE_PASSWORD" default:"changeit"`
}

// KerberosArgs contains Kerberos authentication configuration.
type KerberosArgs struct {
	// UserPrincipal is the Kerberos user principal
	UserPrincipal string `envconfig:"PLUGIN_KERBEROS_USER_PRINCIPAL"`

	// Password is the Kerberos password (mutually exclusive with KeytabFilePath)
	Password string `envconfig:"PLUGIN_KERBEROS_PASSWORD"`

	// KeytabFilePath is the path to the Kerberos keytab file
	KeytabFilePath string `envconfig:"PLUGIN_KERBEROS_KEYTAB_FILE_PATH"`
}
