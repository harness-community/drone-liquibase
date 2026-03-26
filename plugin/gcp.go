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
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	stsTokenURL = "https://sts.googleapis.com/v1/token"

	gcpAudienceFormat                    = "//iam.googleapis.com/projects/%s/locations/global/workloadIdentityPools/%s/providers/%s"
	gcpScopeURL                          = "https://www.googleapis.com/auth/cloud-platform"
	gcpGrantTypeTokenExchange            = "urn:ietf:params:oauth:grant-type:token-exchange"
	gcpTokenTypeIDToken                  = "urn:ietf:params:oauth:token-type:id_token"
	gcpTokenTypeAccessToken              = "urn:ietf:params:oauth:token-type:access_token"
	gcpServiceAccountImpersonationURLFmt = "https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/%s:generateAccessToken"

	spannerURLPrefix  = "jdbc:cloudspanner:"
	postgresURLPrefix = "jdbc:postgresql:"

	gcpServiceAccountSuffix = ".gserviceaccount.com"
)

// stsTokenResponse represents the response from the STS token exchange endpoint.
type stsTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// iamAccessTokenResponse represents the response from IAM generateAccessToken.
type iamAccessTokenResponse struct {
	AccessToken string `json:"accessToken"`
	ExpireTime  string `json:"expireTime"`
}

// SetupGCPOIDCAuth exchanges an OIDC ID token for a GCP access token via
// Workload Identity Federation, then configures Liquibase credentials based
// on the target database type:
//   - Spanner: appends ;oauthToken=<token> to the JDBC URL
//   - CloudSQL PostgreSQL: sets username (email without .gserviceaccount.com) + password
//   - CloudSQL MySQL: sets username (full service account email) + password
func SetupGCPOIDCAuth(args GCPOIDCArgs) (cleanup func(), err error) {
	if args.OIDCIDToken == "" {
		return nil, nil
	}

	if args.ProjectID == "" || args.WorkloadPoolID == "" || args.ProviderID == "" || args.ServiceAccountEmail == "" {
		return nil, fmt.Errorf("GCP OIDC auth requires project_id, workload_pool_id, provider_id, and service_account_email")
	}

	logrus.Info("Setting up GCP OIDC Workload Identity Federation authentication...")

	// Step 1: Exchange OIDC ID token for a federated (STS) token
	federatedToken, err := getFederatedToken(args)
	if err != nil {
		return nil, fmt.Errorf("failed to get federated token: %w", err)
	}
	logrus.Info("Successfully obtained federated token via STS token exchange")

	// Step 2: Exchange federated token for a GCP access token via service account impersonation
	accessToken, err := getGCPAccessToken(federatedToken, args.ServiceAccountEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to get GCP access token: %w", err)
	}
	logrus.Info("Successfully obtained GCP access token via service account impersonation")

	// Step 3: Configure Liquibase credentials based on database type
	liquibaseURL := os.Getenv("PLUGIN_LIQUIBASE_URL")
	var envVarsSet []string

	if isSpannerURL(liquibaseURL) {
		// Spanner does not support username/password.
		// Pass the access token via the oauthToken URL property.
		cleanURL := strings.TrimRight(liquibaseURL, ";?")
		os.Setenv("PLUGIN_LIQUIBASE_URL", cleanURL+";oauthToken="+accessToken)
		envVarsSet = append(envVarsSet, "PLUGIN_LIQUIBASE_URL")
		logrus.Info("Configured Spanner OIDC auth via oauthToken URL property")
	} else {
		// CloudSQL MySQL/PostgreSQL use the access token as the JDBC password
		// and the service account email as the username.
		username := args.ServiceAccountEmail
		if isPostgresURL(liquibaseURL) {
			username = strings.Replace(username, gcpServiceAccountSuffix, "", 1)
		}
		os.Setenv("PLUGIN_LIQUIBASE_USERNAME", username)
		os.Setenv("PLUGIN_LIQUIBASE_PASSWORD", accessToken)
		envVarsSet = append(envVarsSet, "PLUGIN_LIQUIBASE_USERNAME", "PLUGIN_LIQUIBASE_PASSWORD")
		logrus.Infof("Configured CloudSQL OIDC auth with username: %s", username)
	}

	cleanup = func() {
		for _, envVar := range envVarsSet {
			os.Unsetenv(envVar)
		}
	}

	return cleanup, nil
}

// isSpannerURL checks if the JDBC URL targets Cloud Spanner.
func isSpannerURL(jdbcURL string) bool {
	return strings.HasPrefix(strings.ToLower(jdbcURL), spannerURLPrefix)
}

// isPostgresURL checks if the JDBC URL targets PostgreSQL.
func isPostgresURL(jdbcURL string) bool {
	return strings.HasPrefix(strings.ToLower(jdbcURL), postgresURLPrefix)
}

// getFederatedToken exchanges an OIDC ID token for a federated token using the GCP STS endpoint.
func getFederatedToken(args GCPOIDCArgs) (string, error) {
	audience := fmt.Sprintf(gcpAudienceFormat, args.ProjectID, args.WorkloadPoolID, args.ProviderID)

	data := url.Values{
		"grant_type":           {gcpGrantTypeTokenExchange},
		"subject_token":        {args.OIDCIDToken},
		"audience":             {audience},
		"scope":                {gcpScopeURL},
		"requested_token_type": {gcpTokenTypeAccessToken},
		"subject_token_type":   {gcpTokenTypeIDToken},
	}

	resp, err := http.PostForm(stsTokenURL, data)
	if err != nil {
		return "", fmt.Errorf("STS token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read STS response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("STS token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp stsTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse STS token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("STS token exchange returned empty access token")
	}

	return tokenResp.AccessToken, nil
}

// getGCPAccessToken exchanges a federated token for a GCP access token by
// impersonating the specified service account.
func getGCPAccessToken(federatedToken, serviceAccountEmail string) (string, error) {
	impersonateURL := fmt.Sprintf(gcpServiceAccountImpersonationURLFmt, serviceAccountEmail)

	reqBody := fmt.Sprintf(`{"scope":["%s"]}`, gcpScopeURL)
	req, err := http.NewRequest(http.MethodPost, impersonateURL, strings.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create impersonation request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+federatedToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("service account impersonation request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read impersonation response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("service account impersonation failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp iamAccessTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse impersonation response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("service account impersonation returned empty access token")
	}

	return tokenResp.AccessToken, nil
}
