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
	"testing"
)

// useTestCredentialDir overrides oidcCredentialDir to a temp directory for testing.
// Returns a restore function.
func useTestCredentialDir(t *testing.T) func() {
	t.Helper()
	origDir := oidcCredentialDir
	oidcCredentialDir = t.TempDir()
	origCreds := os.Getenv(envGoogleApplicationCredentials)
	return func() {
		oidcCredentialDir = origDir
		if origCreds != "" {
			os.Setenv(envGoogleApplicationCredentials, origCreds)
		} else {
			os.Unsetenv(envGoogleApplicationCredentials)
		}
	}
}

func TestSetupGCPOIDCAuthNoToken(t *testing.T) {
	args := GCPOIDCArgs{
		OIDCIDToken: "", // Not configured
	}

	modifiedURL, cleanup, err := SetupGCPOIDCAuth(args, "")
	if err != nil {
		t.Errorf("SetupGCPOIDCAuth() with no token should not error, got: %v", err)
	}
	if cleanup != nil {
		t.Error("cleanup should be nil when OIDC token not provided")
	}
	if modifiedURL != "" {
		t.Errorf("modifiedURL should be empty, got %q", modifiedURL)
	}
}

func TestSetupGCPOIDCAuthMissingRequiredFields(t *testing.T) {
	tests := []struct {
		name string
		args GCPOIDCArgs
	}{
		{
			name: "missing project_id",
			args: GCPOIDCArgs{
				OIDCIDToken:         "token",
				ProjectID:           "",
				WorkloadPoolID:      "pool",
				ProviderID:          "provider",
				ServiceAccountEmail: "sa@proj.iam.gserviceaccount.com",
			},
		},
		{
			name: "missing workload_pool_id",
			args: GCPOIDCArgs{
				OIDCIDToken:         "token",
				ProjectID:           "123",
				WorkloadPoolID:      "",
				ProviderID:          "provider",
				ServiceAccountEmail: "sa@proj.iam.gserviceaccount.com",
			},
		},
		{
			name: "missing provider_id",
			args: GCPOIDCArgs{
				OIDCIDToken:         "token",
				ProjectID:           "123",
				WorkloadPoolID:      "pool",
				ProviderID:          "",
				ServiceAccountEmail: "sa@proj.iam.gserviceaccount.com",
			},
		},
		{
			name: "missing service_account_email",
			args: GCPOIDCArgs{
				OIDCIDToken:         "token",
				ProjectID:           "123",
				WorkloadPoolID:      "pool",
				ProviderID:          "provider",
				ServiceAccountEmail: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := SetupGCPOIDCAuth(tt.args, "")
			if err == nil {
				t.Error("SetupGCPOIDCAuth() should error when required fields are missing")
			}
			expectedMsg := "GCP OIDC auth requires project_id, workload_pool_id, provider_id, and service_account_email"
			if err.Error() != expectedMsg {
				t.Errorf("Error message = %q, want %q", err.Error(), expectedMsg)
			}
		})
	}
}

// verifyCredentialFiles checks that the OIDC token file and credential config
// were written correctly and GOOGLE_APPLICATION_CREDENTIALS is set.
func verifyCredentialFiles(t *testing.T, args GCPOIDCArgs) {
	t.Helper()

	tokenFile := filepath.Join(oidcCredentialDir, "oidc-token")
	credFile := filepath.Join(oidcCredentialDir, "gcp-oidc-credentials.json")

	// Verify GOOGLE_APPLICATION_CREDENTIALS points to credential config
	credPath := os.Getenv(envGoogleApplicationCredentials)
	if credPath != credFile {
		t.Errorf("GOOGLE_APPLICATION_CREDENTIALS = %q, want %q", credPath, credFile)
	}

	// Verify OIDC token file content
	tokenData, err := os.ReadFile(tokenFile)
	if err != nil {
		t.Fatalf("failed to read OIDC token file: %v", err)
	}
	if string(tokenData) != args.OIDCIDToken {
		t.Errorf("OIDC token file content = %q, want %q", string(tokenData), args.OIDCIDToken)
	}

	// Verify credential config structure
	configData, err := os.ReadFile(credFile)
	if err != nil {
		t.Fatalf("failed to read credential config file: %v", err)
	}
	var config externalAccountConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		t.Fatalf("failed to parse credential config: %v", err)
	}
	if config.Type != externalAccountType {
		t.Errorf("config.Type = %q, want %q", config.Type, externalAccountType)
	}
	if config.TokenURL != stsTokenURL {
		t.Errorf("config.TokenURL = %q, want %q", config.TokenURL, stsTokenURL)
	}
	if config.SubjectTokenType != gcpTokenTypeIDToken {
		t.Errorf("config.SubjectTokenType = %q, want %q", config.SubjectTokenType, gcpTokenTypeIDToken)
	}
	if config.CredentialSource.File != tokenFile {
		t.Errorf("config.CredentialSource.File = %q, want %q", config.CredentialSource.File, tokenFile)
	}
	expectedAudience := fmt.Sprintf(gcpAudienceFormat, args.ProjectID, args.WorkloadPoolID, args.ProviderID)
	if config.Audience != expectedAudience {
		t.Errorf("config.Audience = %q, want %q", config.Audience, expectedAudience)
	}
	expectedImpersonationURL := fmt.Sprintf(gcpServiceAccountImpersonationURLFmt, args.ServiceAccountEmail)
	if config.ServiceAccountImpersonationURL != expectedImpersonationURL {
		t.Errorf("config.ServiceAccountImpersonationURL = %q, want %q", config.ServiceAccountImpersonationURL, expectedImpersonationURL)
	}
}

func TestSetupGCPOIDCAuthCredentialConfig(t *testing.T) {
	testArgs := GCPOIDCArgs{
		OIDCIDToken:         "fake-oidc-id-token",
		ProjectID:           "123456",
		WorkloadPoolID:      "my-pool",
		ProviderID:          "my-provider",
		ServiceAccountEmail: "sa@proj.iam.gserviceaccount.com",
	}

	tests := []struct {
		name        string
		url         string
		wantURLMod  bool
		wantURL     string
	}{
		{
			name:       "spanner URL - credential config only, no URL modification",
			url:        "jdbc:cloudspanner:/projects/my-project/instances/my-instance/databases/my-db",
			wantURLMod: false,
		},
		{
			name:       "empty URL - credential config only",
			url:        "",
			wantURLMod: false,
		},
		{
			name:       "direct IP postgres - credential config only, no URL modification",
			url:        "jdbc:postgresql://10.0.0.1:5432/mydb",
			wantURLMod: false,
		},
		{
			name:       "cloudsql postgres with socketFactory - URL gets user appended",
			url:        "jdbc:postgresql:///mydb?cloudSqlInstance=proj:us-central1:inst&socketFactory=com.google.cloud.sql.postgres.SocketFactory&enableIamAuth=true",
			wantURLMod: true,
			wantURL:    "jdbc:postgresql:///mydb?cloudSqlInstance=proj:us-central1:inst&socketFactory=com.google.cloud.sql.postgres.SocketFactory&enableIamAuth=true&user=sa@proj.iam",
		},
		{
			name:       "cloudsql mysql with socketFactory - URL gets user appended",
			url:        "jdbc:mysql:///mydb?cloudSqlInstance=proj:us-central1:inst&socketFactory=com.google.cloud.sql.mysql.SocketFactory&enableIamAuth=true",
			wantURLMod: true,
			wantURL:    "jdbc:mysql:///mydb?cloudSqlInstance=proj:us-central1:inst&socketFactory=com.google.cloud.sql.mysql.SocketFactory&enableIamAuth=true&user=sa",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := useTestCredentialDir(t)
			defer restore()

			modifiedURL, cleanup, err := SetupGCPOIDCAuth(testArgs, tt.url)
			if err != nil {
				t.Fatalf("SetupGCPOIDCAuth() error = %v", err)
			}
			if cleanup == nil {
				t.Fatal("cleanup should not be nil")
			}
			defer cleanup()

			// Verify credential files are written for ALL cases
			verifyCredentialFiles(t, testArgs)

			// Verify URL modification
			if tt.wantURLMod {
				if modifiedURL != tt.wantURL {
					t.Errorf("modifiedURL = %q, want %q", modifiedURL, tt.wantURL)
				}
			} else {
				if modifiedURL != "" {
					t.Errorf("modifiedURL should be empty, got %q", modifiedURL)
				}
			}
		})
	}
}

func TestSetupGCPOIDCAuthCleanup(t *testing.T) {
	restore := useTestCredentialDir(t)
	defer restore()

	args := GCPOIDCArgs{
		OIDCIDToken:         "fake-oidc-id-token",
		ProjectID:           "123456",
		WorkloadPoolID:      "my-pool",
		ProviderID:          "my-provider",
		ServiceAccountEmail: "sa@proj.iam.gserviceaccount.com",
	}

	_, cleanup, err := SetupGCPOIDCAuth(args, "")
	if err != nil {
		t.Fatalf("SetupGCPOIDCAuth() error = %v", err)
	}

	cleanup()

	tokenFile := filepath.Join(oidcCredentialDir, "oidc-token")
	credFile := filepath.Join(oidcCredentialDir, "gcp-oidc-credentials.json")
	if _, err := os.Stat(tokenFile); !os.IsNotExist(err) {
		t.Error("OIDC token file should be removed after cleanup")
	}
	if _, err := os.Stat(credFile); !os.IsNotExist(err) {
		t.Error("credential config file should be removed after cleanup")
	}
	if val := os.Getenv(envGoogleApplicationCredentials); val != "" {
		t.Errorf("GOOGLE_APPLICATION_CREDENTIALS should be unset after cleanup, got %q", val)
	}
}

func TestSetupGCPOIDCAuthCloudSQLPostgresUser(t *testing.T) {
	restore := useTestCredentialDir(t)
	defer restore()

	args := GCPOIDCArgs{
		OIDCIDToken:         "fake-oidc-id-token",
		ProjectID:           "123456",
		WorkloadPoolID:      "my-pool",
		ProviderID:          "my-provider",
		ServiceAccountEmail: "my-sa@project-id.iam.gserviceaccount.com",
	}

	pgURL := "jdbc:postgresql:///mydb?cloudSqlInstance=project:region:instance&socketFactory=com.google.cloud.sql.postgres.SocketFactory&enableIamAuth=true"
	modifiedURL, cleanup, err := SetupGCPOIDCAuth(args, pgURL)
	if err != nil {
		t.Fatalf("SetupGCPOIDCAuth() error = %v", err)
	}
	defer cleanup()

	wantURL := pgURL + "&user=my-sa@project-id.iam"
	if modifiedURL != wantURL {
		t.Errorf("modifiedURL = %q, want %q", modifiedURL, wantURL)
	}
}

func TestSetupGCPOIDCAuthCloudSQLPostgresUserReplace(t *testing.T) {
	restore := useTestCredentialDir(t)
	defer restore()

	args := GCPOIDCArgs{
		OIDCIDToken:         "fake-oidc-id-token",
		ProjectID:           "123456",
		WorkloadPoolID:      "my-pool",
		ProviderID:          "my-provider",
		ServiceAccountEmail: "my-sa@project-id.iam.gserviceaccount.com",
	}

	// URL already has user=some-user, should be replaced
	pgURL := "jdbc:postgresql:///mydb?cloudSqlInstance=project:region:instance&socketFactory=com.google.cloud.sql.postgres.SocketFactory&enableIamAuth=true&user=some-user"
	modifiedURL, cleanup, err := SetupGCPOIDCAuth(args, pgURL)
	if err != nil {
		t.Fatalf("SetupGCPOIDCAuth() error = %v", err)
	}
	defer cleanup()

	wantURL := "jdbc:postgresql:///mydb?cloudSqlInstance=project:region:instance&socketFactory=com.google.cloud.sql.postgres.SocketFactory&enableIamAuth=true&user=my-sa@project-id.iam"
	if modifiedURL != wantURL {
		t.Errorf("modifiedURL = %q, want %q", modifiedURL, wantURL)
	}
}

func TestSetupGCPOIDCAuthCloudSQLMySQLUser(t *testing.T) {
	restore := useTestCredentialDir(t)
	defer restore()

	args := GCPOIDCArgs{
		OIDCIDToken:         "fake-oidc-id-token",
		ProjectID:           "123456",
		WorkloadPoolID:      "my-pool",
		ProviderID:          "my-provider",
		ServiceAccountEmail: "my-sa@project-id.iam.gserviceaccount.com",
	}

	mysqlURL := "jdbc:mysql:///mydb?cloudSqlInstance=project:region:instance&socketFactory=com.google.cloud.sql.mysql.SocketFactory&enableIamAuth=true"
	modifiedURL, cleanup, err := SetupGCPOIDCAuth(args, mysqlURL)
	if err != nil {
		t.Fatalf("SetupGCPOIDCAuth() error = %v", err)
	}
	defer cleanup()

	wantURL := mysqlURL + "&user=my-sa"
	if modifiedURL != wantURL {
		t.Errorf("modifiedURL = %q, want %q", modifiedURL, wantURL)
	}
}

func TestSetupGCPOIDCAuthNoSocketFactory(t *testing.T) {
	restore := useTestCredentialDir(t)
	defer restore()

	args := GCPOIDCArgs{
		OIDCIDToken:         "fake-oidc-id-token",
		ProjectID:           "123456",
		WorkloadPoolID:      "my-pool",
		ProviderID:          "my-provider",
		ServiceAccountEmail: "my-sa@project-id.iam.gserviceaccount.com",
	}

	// Direct IP URL without socketFactory — no URL modification
	pgURL := "jdbc:postgresql://1.2.3.4:5432/mydb"
	modifiedURL, cleanup, err := SetupGCPOIDCAuth(args, pgURL)
	if err != nil {
		t.Fatalf("SetupGCPOIDCAuth() error = %v", err)
	}
	defer cleanup()

	if modifiedURL != "" {
		t.Errorf("modifiedURL should be empty for direct IP URL, got %q", modifiedURL)
	}
}

func TestIsCloudSQLSocketFactoryURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"jdbc:postgresql:///db?socketFactory=com.google.cloud.sql.postgres.SocketFactory", true},
		{"jdbc:mysql:///db?cloudSqlInstance=proj:region:inst&socketFactory=com.google.cloud.sql.mysql.SocketFactory", true},
		{"jdbc:postgresql:///db?socketFactory=com.google.cloud.sql.postgres.SocketFactory&enableIamAuth=true", true},
		{"jdbc:postgresql://1.2.3.4:5432/db", false},
		{"jdbc:cloudspanner:/projects/proj/instances/inst/databases/db", false},
		{"", false},
		// Ensure partial match doesn't trigger (socketFactoryClass vs socketFactory=)
		{"jdbc:postgresql:///db?socketFactoryClass=something", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			if got := isCloudSQLSocketFactoryURL(tt.url); got != tt.want {
				t.Errorf("isCloudSQLSocketFactoryURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestBuildCloudSQLIAMUser(t *testing.T) {
	tests := []struct {
		name  string
		url   string
		email string
		want  string
	}{
		{
			name:  "postgres",
			url:   "jdbc:postgresql:///db?socketFactory=com.google.cloud.sql.postgres.SocketFactory",
			email: "my-sa@project-id.iam.gserviceaccount.com",
			want:  "my-sa@project-id.iam",
		},
		{
			name:  "mysql",
			url:   "jdbc:mysql:///db?socketFactory=com.google.cloud.sql.mysql.SocketFactory",
			email: "my-sa@project-id.iam.gserviceaccount.com",
			want:  "my-sa",
		},
		{
			name:  "postgres uppercase",
			url:   "JDBC:POSTGRESQL:///db?socketFactory=test",
			email: "sa@proj.iam.gserviceaccount.com",
			want:  "sa@proj.iam",
		},
		{
			name:  "mysql uppercase",
			url:   "JDBC:MYSQL:///db?socketFactory=test",
			email: "sa@proj.iam.gserviceaccount.com",
			want:  "sa",
		},
		{
			name:  "unknown db type defaults to sa name",
			url:   "jdbc:sqlserver:///db?socketFactory=test",
			email: "my-sa@project-id.iam.gserviceaccount.com",
			want:  "my-sa",
		},
		{
			name:  "postgres email without gserviceaccount suffix",
			url:   "jdbc:postgresql:///db?socketFactory=test",
			email: "user@example.com",
			want:  "user@example.com",
		},
		{
			name:  "mysql email without @ symbol",
			url:   "jdbc:mysql:///db?socketFactory=test",
			email: "localuser",
			want:  "localuser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildCloudSQLIAMUser(tt.url, tt.email); got != tt.want {
				t.Errorf("buildCloudSQLIAMUser() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetURLProperty(t *testing.T) {
	tests := []struct {
		name  string
		url   string
		key   string
		value string
		want  string
	}{
		{
			name:  "append with existing params",
			url:   "jdbc:postgresql:///db?foo=bar",
			key:   "user",
			value: "sa@proj.iam",
			want:  "jdbc:postgresql:///db?foo=bar&user=sa@proj.iam",
		},
		{
			name:  "append without params",
			url:   "jdbc:postgresql:///db",
			key:   "user",
			value: "sa@proj.iam",
			want:  "jdbc:postgresql:///db?user=sa@proj.iam",
		},
		{
			name:  "replace existing value",
			url:   "jdbc:postgresql:///db?user=old&foo=bar",
			key:   "user",
			value: "new",
			want:  "jdbc:postgresql:///db?user=new&foo=bar",
		},
		{
			name:  "replace last param",
			url:   "jdbc:postgresql:///db?foo=bar&user=old",
			key:   "user",
			value: "new",
			want:  "jdbc:postgresql:///db?foo=bar&user=new",
		},
		{
			name:  "replace first param",
			url:   "jdbc:postgresql:///db?user=old&foo=bar",
			key:   "user",
			value: "new",
			want:  "jdbc:postgresql:///db?user=new&foo=bar",
		},
		{
			name:  "does not match different property name",
			url:   "jdbc:postgresql:///db?username=admin",
			key:   "user",
			value: "sa@proj.iam",
			want:  "jdbc:postgresql:///db?username=admin&user=sa@proj.iam",
		},
		{
			name:  "matches substring in property name",
			url:   "jdbc:postgresql:///db?foouser=bar",
			key:   "user",
			value: "sa@proj.iam",
			want:  "jdbc:postgresql:///db?foouser=bar&user=sa@proj.iam",
		},
		{
			name:  "replace only param",
			url:   "jdbc:postgresql:///db?user=old",
			key:   "user",
			value: "new",
			want:  "jdbc:postgresql:///db?user=new",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := setURLProperty(tt.url, tt.key, tt.value); got != tt.want {
				t.Errorf("setURLProperty() = %q, want %q", got, tt.want)
			}
		})
	}
}
