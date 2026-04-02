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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// oidcCredentialDir is the directory for OIDC credential files.
// Overridable in tests.
var oidcCredentialDir = defaultOIDCCredentialDir

const (
	stsTokenURL                          = "https://sts.googleapis.com/v1/token"
	gcpAudienceFormat                    = "//iam.googleapis.com/projects/%s/locations/global/workloadIdentityPools/%s/providers/%s"
	gcpTokenTypeIDToken                  = "urn:ietf:params:oauth:token-type:id_token"
	gcpServiceAccountImpersonationURLFmt = "https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/%s:generateAccessToken"

	envGoogleApplicationCredentials = "GOOGLE_APPLICATION_CREDENTIALS"
	envPluginLiquibaseURL           = "PLUGIN_LIQUIBASE_URL"

	socketFactoryProperty   = "socketFactory"
	postgresURLPrefix       = "jdbc:postgresql:"
	gcpServiceAccountSuffix = ".gserviceaccount.com"

	defaultOIDCCredentialDir = "/harness"

	externalAccountType = "external_account"
)

// externalAccountConfig represents the GCP external account credential configuration
// used for Workload Identity Federation via OIDC.
type externalAccountConfig struct {
	Type                           string           `json:"type"`
	Audience                       string           `json:"audience"`
	SubjectTokenType               string           `json:"subject_token_type"`
	TokenURL                       string           `json:"token_url"`
	CredentialSource               credentialSource `json:"credential_source"`
	ServiceAccountImpersonationURL string           `json:"service_account_impersonation_url"`
}

type credentialSource struct {
	File string `json:"file"`
}

// SetupGCPOIDCAuth configures GCP OIDC Workload Identity Federation by writing
// an external account credential config file and setting GOOGLE_APPLICATION_CREDENTIALS.
// The GCP-aware JDBC drivers (Spanner, CloudSQL Socket Factory) handle token
// exchange and authentication automatically via Application Default Credentials (ADC).
//
// For CloudSQL URLs with socketFactory, it also sets the IAM-appropriate user
// property in the JDBC URL and returns the modified URL.
//
// Returns the modified JDBC URL (empty if unchanged) and a cleanup function.
func SetupGCPOIDCAuth(args GCPOIDCArgs, liquibaseURL string) (modifiedURL string, cleanup func(), err error) {
	if args.OIDCIDToken == "" {
		return "", nil, nil
	}

	if args.ProjectID == "" || args.WorkloadPoolID == "" || args.ProviderID == "" || args.ServiceAccountEmail == "" {
		return "", nil, fmt.Errorf("GCP OIDC auth requires project_id, workload_pool_id, provider_id, and service_account_email")
	}

	logrus.Info("Setting up GCP OIDC Workload Identity Federation authentication...")

	tokenFile := filepath.Join(oidcCredentialDir, "oidc-token")
	credFile := filepath.Join(oidcCredentialDir, "gcp-oidc-credentials.json")

	// Ensure the directory exists (non-root containers may not have it)
	if err := os.MkdirAll(oidcCredentialDir, 0755); err != nil {
		return "", nil, fmt.Errorf("failed to create OIDC credential directory: %w", err)
	}

	// Write the OIDC ID token to a file for the credential source
	if err := os.WriteFile(tokenFile, []byte(args.OIDCIDToken), 0600); err != nil {
		return "", nil, fmt.Errorf("failed to write OIDC token file: %w", err)
	}

	// Build external account credential config
	config := externalAccountConfig{
		Type:             externalAccountType,
		Audience:         fmt.Sprintf(gcpAudienceFormat, args.ProjectID, args.WorkloadPoolID, args.ProviderID),
		SubjectTokenType: gcpTokenTypeIDToken,
		TokenURL:         stsTokenURL,
		CredentialSource: credentialSource{
			File: tokenFile,
		},
		ServiceAccountImpersonationURL: fmt.Sprintf(gcpServiceAccountImpersonationURLFmt, args.ServiceAccountEmail),
	}

	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		os.Remove(tokenFile)
		return "", nil, fmt.Errorf("failed to marshal credential config: %w", err)
	}

	if err := os.WriteFile(credFile, configJSON, 0600); err != nil {
		os.Remove(tokenFile)
		return "", nil, fmt.Errorf("failed to write credential config file: %w", err)
	}

	os.Setenv(envGoogleApplicationCredentials, credFile)
	logrus.Info("Configured GCP OIDC auth via GOOGLE_APPLICATION_CREDENTIALS (external account)")

	cleanup = func() {
		os.Remove(tokenFile)
		os.Remove(credFile)
		os.Unsetenv(envGoogleApplicationCredentials)
	}

	// For CloudSQL URLs with socketFactory, set the IAM-appropriate user property
	if isCloudSQLSocketFactoryURL(liquibaseURL) {
		iamUser := buildCloudSQLIAMUser(liquibaseURL, args.ServiceAccountEmail)
		modifiedURL = setURLProperty(liquibaseURL, "user", iamUser)
		logrus.Infof("Configured CloudSQL IAM user in JDBC URL: %s", iamUser)
	}

	return modifiedURL, cleanup, nil
}

// isCloudSQLSocketFactoryURL checks if the JDBC URL uses Cloud SQL Socket Factory.
func isCloudSQLSocketFactoryURL(jdbcURL string) bool {
	return strings.Contains(jdbcURL, socketFactoryProperty+"=")
}

// buildCloudSQLIAMUser derives the IAM database username from the service account email
// based on the database type:
//   - PostgreSQL: sa-name@project-id.iam (strips .gserviceaccount.com)
//   - Default (MySQL and others): sa-name (part before @)
func buildCloudSQLIAMUser(jdbcURL, serviceAccountEmail string) string {
	lowerURL := strings.ToLower(jdbcURL)
	if strings.HasPrefix(lowerURL, postgresURLPrefix) {
		// PostgreSQL: strip .gserviceaccount.com
		return strings.TrimSuffix(serviceAccountEmail, gcpServiceAccountSuffix)
	}
	// Default (MySQL and others): use only the service account name (before @)
	if idx := strings.Index(serviceAccountEmail, "@"); idx != -1 {
		return serviceAccountEmail[:idx]
	}
	return serviceAccountEmail
}

// setURLProperty adds or replaces a property in a JDBC URL.
// Handles both ? and & separators.
func setURLProperty(jdbcURL, key, value string) string {
	prop := key + "="

	// If property exists, replace its value
	if idx := strings.Index(jdbcURL, prop); idx != -1 {
		// Find the end of the current value (next & or end of string)
		valueStart := idx + len(prop)
		valueEnd := strings.Index(jdbcURL[valueStart:], "&")
		if valueEnd == -1 {
			return jdbcURL[:valueStart] + value
		}
		return jdbcURL[:valueStart] + value + jdbcURL[valueStart+valueEnd:]
	}

	// Append new property
	separator := "&"
	if !strings.Contains(jdbcURL, "?") {
		separator = "?"
	}
	return jdbcURL + separator + key + "=" + value
}
