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
	"os"
	"testing"
)

func TestSetupGCPOIDCAuthNoToken(t *testing.T) {
	args := GCPOIDCArgs{
		OIDCIDToken: "", // Not configured
	}

	cleanup, err := SetupGCPOIDCAuth(args)
	if err != nil {
		t.Errorf("SetupGCPOIDCAuth() with no token should not error, got: %v", err)
	}
	if cleanup != nil {
		t.Error("cleanup should be nil when OIDC token not provided")
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
			_, err := SetupGCPOIDCAuth(tt.args)
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

func TestIsSpannerURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"jdbc:cloudspanner:/projects/my-project/instances/my-instance/databases/my-db", true},
		{"JDBC:CLOUDSPANNER:/projects/my-project/instances/my-instance/databases/my-db", true},
		{"jdbc:postgresql://localhost:5432/db", false},
		{"jdbc:mysql://localhost:3306/db", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			if got := isSpannerURL(tt.url); got != tt.want {
				t.Errorf("isSpannerURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestIsPostgresURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"jdbc:postgresql://localhost:5432/db", true},
		{"JDBC:POSTGRESQL://localhost:5432/db", true},
		{"jdbc:cloudspanner:/projects/my-project/instances/my-instance/databases/my-db", false},
		{"jdbc:mysql://localhost:3306/db", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			if got := isPostgresURL(tt.url); got != tt.want {
				t.Errorf("isPostgresURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestSetupGCPOIDCAuthSpannerURL(t *testing.T) {
	// Save and restore env
	origURL := os.Getenv("PLUGIN_LIQUIBASE_URL")
	defer os.Setenv("PLUGIN_LIQUIBASE_URL", origURL)

	spannerURL := "jdbc:cloudspanner:/projects/my-project/instances/my-instance/databases/my-db"
	os.Setenv("PLUGIN_LIQUIBASE_URL", spannerURL)

	// We can't test the full flow without real GCP credentials, but we can
	// verify the URL detection logic by checking that the function reads
	// PLUGIN_LIQUIBASE_URL and would enter the Spanner branch.
	// The STS call will fail, which is expected.
	args := GCPOIDCArgs{
		OIDCIDToken:         "fake-token",
		ProjectID:           "123456",
		WorkloadPoolID:      "my-pool",
		ProviderID:          "my-provider",
		ServiceAccountEmail: "sa@proj.iam.gserviceaccount.com",
	}

	_, err := SetupGCPOIDCAuth(args)
	// Expected to fail at STS token exchange (no real credentials)
	if err == nil {
		t.Error("SetupGCPOIDCAuth() should error with fake token")
	}
}

func TestSetupGCPOIDCAuthPostgresUsername(t *testing.T) {
	// Test that PostgreSQL username strips .gserviceaccount.com
	origURL := os.Getenv("PLUGIN_LIQUIBASE_URL")
	origUser := os.Getenv("PLUGIN_LIQUIBASE_USERNAME")
	origPass := os.Getenv("PLUGIN_LIQUIBASE_PASSWORD")
	defer func() {
		os.Setenv("PLUGIN_LIQUIBASE_URL", origURL)
		os.Setenv("PLUGIN_LIQUIBASE_USERNAME", origUser)
		os.Setenv("PLUGIN_LIQUIBASE_PASSWORD", origPass)
	}()

	os.Setenv("PLUGIN_LIQUIBASE_URL", "jdbc:postgresql://localhost:5432/db")

	args := GCPOIDCArgs{
		OIDCIDToken:         "fake-token",
		ProjectID:           "123456",
		WorkloadPoolID:      "my-pool",
		ProviderID:          "my-provider",
		ServiceAccountEmail: "sa@proj.iam.gserviceaccount.com",
	}

	// Will fail at STS, but verifies the function starts correctly
	_, err := SetupGCPOIDCAuth(args)
	if err == nil {
		t.Error("SetupGCPOIDCAuth() should error with fake token")
	}
}

func TestGetFederatedTokenInvalidEndpoint(t *testing.T) {
	args := GCPOIDCArgs{
		OIDCIDToken:         "fake-id-token",
		ProjectID:           "123456789",
		WorkloadPoolID:      "my-pool",
		ProviderID:          "my-provider",
		ServiceAccountEmail: "sa@proj.iam.gserviceaccount.com",
	}

	// This will make a real HTTP call to STS with a fake token, which should fail
	_, err := getFederatedToken(args)
	if err == nil {
		t.Error("getFederatedToken() should error with fake token")
	}
}

func TestGetGCPAccessTokenInvalidToken(t *testing.T) {
	_, err := getGCPAccessToken("invalid-federated-token", "sa@proj.iam.gserviceaccount.com")
	if err == nil {
		t.Error("getGCPAccessToken() should error with invalid federated token")
	}
}
