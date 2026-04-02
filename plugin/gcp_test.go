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
	"os"
	"testing"
)

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

func TestSetupGCPOIDCAuthCredentialConfig(t *testing.T) {
	origCreds := os.Getenv(envGoogleApplicationCredentials)
	defer func() {
		if origCreds != "" {
			os.Setenv(envGoogleApplicationCredentials, origCreds)
		} else {
			os.Unsetenv(envGoogleApplicationCredentials)
		}
	}()

	args := GCPOIDCArgs{
		OIDCIDToken:         "fake-oidc-id-token",
		ProjectID:           "123456",
		WorkloadPoolID:      "my-pool",
		ProviderID:          "my-provider",
		ServiceAccountEmail: "sa@proj.iam.gserviceaccount.com",
	}

	// Spanner URL — no socketFactory, so no URL modification
	spannerURL := "jdbc:cloudspanner:/projects/my-project/instances/my-instance/databases/my-db"
	modifiedURL, cleanup, err := SetupGCPOIDCAuth(args, spannerURL)
	if err != nil {
		t.Fatalf("SetupGCPOIDCAuth() error = %v", err)
	}
	defer cleanup()

	if modifiedURL != "" {
		t.Errorf("modifiedURL should be empty for Spanner, got %q", modifiedURL)
	}

	// Verify credential config was written
	credPath := os.Getenv(envGoogleApplicationCredentials)
	if credPath != oidcCredentialFilePath {
		t.Errorf("GOOGLE_APPLICATION_CREDENTIALS = %q, want %q", credPath, oidcCredentialFilePath)
	}

	configData, err := os.ReadFile(oidcCredentialFilePath)
	if err != nil {
		t.Fatalf("failed to read credential config: %v", err)
	}
	var config externalAccountConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		t.Fatalf("failed to parse credential config: %v", err)
	}
	if config.Type != externalAccountType {
		t.Errorf("config.Type = %q, want %q", config.Type, externalAccountType)
	}
}

func TestSetupGCPOIDCAuthCleanup(t *testing.T) {
	origCreds := os.Getenv(envGoogleApplicationCredentials)
	defer func() {
		if origCreds != "" {
			os.Setenv(envGoogleApplicationCredentials, origCreds)
		} else {
			os.Unsetenv(envGoogleApplicationCredentials)
		}
	}()

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

	if _, err := os.Stat(oidcTokenFilePath); !os.IsNotExist(err) {
		t.Error("OIDC token file should be removed after cleanup")
	}
	if _, err := os.Stat(oidcCredentialFilePath); !os.IsNotExist(err) {
		t.Error("credential config file should be removed after cleanup")
	}
	if val := os.Getenv(envGoogleApplicationCredentials); val != "" {
		t.Errorf("GOOGLE_APPLICATION_CREDENTIALS should be unset after cleanup, got %q", val)
	}
}

func TestSetupGCPOIDCAuthCloudSQLPostgresUser(t *testing.T) {
	origCreds := os.Getenv(envGoogleApplicationCredentials)
	defer func() {
		if origCreds != "" {
			os.Setenv(envGoogleApplicationCredentials, origCreds)
		} else {
			os.Unsetenv(envGoogleApplicationCredentials)
		}
	}()

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
	origCreds := os.Getenv(envGoogleApplicationCredentials)
	defer func() {
		if origCreds != "" {
			os.Setenv(envGoogleApplicationCredentials, origCreds)
		} else {
			os.Unsetenv(envGoogleApplicationCredentials)
		}
	}()

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
	origCreds := os.Getenv(envGoogleApplicationCredentials)
	defer func() {
		if origCreds != "" {
			os.Setenv(envGoogleApplicationCredentials, origCreds)
		} else {
			os.Unsetenv(envGoogleApplicationCredentials)
		}
	}()

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
	origCreds := os.Getenv(envGoogleApplicationCredentials)
	defer func() {
		if origCreds != "" {
			os.Setenv(envGoogleApplicationCredentials, origCreds)
		} else {
			os.Unsetenv(envGoogleApplicationCredentials)
		}
	}()

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := setURLProperty(tt.url, tt.key, tt.value); got != tt.want {
				t.Errorf("setURLProperty() = %q, want %q", got, tt.want)
			}
		})
	}
}
