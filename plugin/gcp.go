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
	mysqlURLPrefix    = "jdbc:mysql:"

	gcpServiceAccountSuffix = ".gserviceaccount.com"

	headerAuthorization = "Authorization"
	headerContentType   = "Content-Type"
	contentTypeJSON     = "application/json"
	bearerTokenFormat   = "Bearer %s"

	stsParamGrantType          = "grant_type"
	stsParamSubjectToken       = "subject_token"
	stsParamAudience           = "audience"
	stsParamScope              = "scope"
	stsParamRequestedTokenType = "requested_token_type"
	stsParamSubjectTokenType   = "subject_token_type"

	envPluginLiquibaseURL      = "PLUGIN_LIQUIBASE_URL"
	envPluginLiquibaseUsername = "PLUGIN_LIQUIBASE_USERNAME"
	envPluginLiquibasePassword = "PLUGIN_LIQUIBASE_PASSWORD"
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

// ModifyGcpOidcAuthOverrides exchanges an OIDC ID token for a GCP access token via
// Workload Identity Federation, then populates envOverrides with Liquibase
// credential env vars based on the target database type:
//   - Spanner: sets PLUGIN_LIQUIBASE_URL with ;oauthToken=<token> appended
//   - CloudSQL PostgreSQL: sets username (email without .gserviceaccount.com) + password
//   - CloudSQL MySQL: sets username (full service account email) + password
func ModifyGcpOidcAuthOverrides(args GCPOIDCArgs, liquibaseURL string, envOverrides map[string]string) error {
	if args.OIDCIDToken == "" {
		return nil
	}

	if args.ProjectID == "" || args.WorkloadPoolID == "" || args.ProviderID == "" || args.ServiceAccountEmail == "" {
		return fmt.Errorf("GCP OIDC auth requires project_id, workload_pool_id, provider_id, and service_account_email")
	}

	logrus.Info("Setting up GCP OIDC Workload Identity Federation authentication...")

	// Step 1: Exchange OIDC ID token for a federated (STS) token
	federatedToken, err := getFederatedToken(args)
	if err != nil {
		return fmt.Errorf("failed to get federated token: %w", err)
	}
	logrus.Info("Successfully obtained federated token via STS token exchange")

	// Step 2: Exchange federated token for a GCP access token via service account impersonation
	accessToken, err := getGCPAccessToken(federatedToken, args.ServiceAccountEmail)
	if err != nil {
		return fmt.Errorf("failed to get GCP access token: %w", err)
	}
	logrus.Info("Successfully obtained GCP access token via service account impersonation")

	// Step 3: Populate auth overrides based on database type
	if isSpannerURL(liquibaseURL) {
		// Spanner does not support username/password.
		// Pass the access token via the oauthToken URL property.
		cleanURL := strings.TrimRight(liquibaseURL, ";?")
		envOverrides[envPluginLiquibaseURL] = cleanURL + ";oauthToken=" + url.QueryEscape(accessToken)
		logrus.Info("Configured Spanner OIDC auth via oauthToken URL property")
	} else if isPostgresURL(liquibaseURL) {
		// CloudSQL PostgreSQL: username is email without .gserviceaccount.com
		username := strings.Replace(args.ServiceAccountEmail, gcpServiceAccountSuffix, "", 1)
		envOverrides[envPluginLiquibaseUsername] = username
		envOverrides[envPluginLiquibasePassword] = accessToken
		logrus.Infof("Configured CloudSQL PostgreSQL OIDC auth with username: %s", username)
	} else if isMySQLURL(liquibaseURL) {
		// CloudSQL MySQL: username is the full service account email
		envOverrides[envPluginLiquibaseUsername] = args.ServiceAccountEmail
		envOverrides[envPluginLiquibasePassword] = accessToken
		logrus.Infof("Configured CloudSQL MySQL OIDC auth with username: %s", args.ServiceAccountEmail)
	} else {
		// Unknown database type: use full service account email as username
		envOverrides[envPluginLiquibaseUsername] = args.ServiceAccountEmail
		envOverrides[envPluginLiquibasePassword] = accessToken
		logrus.Warnf("Unknown JDBC URL type, using full service account email as username: %s", args.ServiceAccountEmail)
	}

	return nil
}

// isSpannerURL checks if the JDBC URL targets Cloud Spanner.
func isSpannerURL(jdbcURL string) bool {
	return strings.HasPrefix(strings.ToLower(jdbcURL), spannerURLPrefix)
}

// isPostgresURL checks if the JDBC URL targets PostgreSQL.
func isPostgresURL(jdbcURL string) bool {
	return strings.HasPrefix(strings.ToLower(jdbcURL), postgresURLPrefix)
}

// isMySQLURL checks if the JDBC URL targets MySQL.
func isMySQLURL(jdbcURL string) bool {
	return strings.HasPrefix(strings.ToLower(jdbcURL), mysqlURLPrefix)
}

// getFederatedToken exchanges an OIDC ID token for a federated token using the GCP STS endpoint.
func getFederatedToken(args GCPOIDCArgs) (string, error) {
	audience := fmt.Sprintf(gcpAudienceFormat, args.ProjectID, args.WorkloadPoolID, args.ProviderID)

	data := url.Values{
		stsParamGrantType:          {gcpGrantTypeTokenExchange},
		stsParamSubjectToken:       {args.OIDCIDToken},
		stsParamAudience:           {audience},
		stsParamScope:              {gcpScopeURL},
		stsParamRequestedTokenType: {gcpTokenTypeAccessToken},
		stsParamSubjectTokenType:   {gcpTokenTypeIDToken},
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

	req.Header.Set(headerAuthorization, fmt.Sprintf(bearerTokenFormat, federatedToken))
	req.Header.Set(headerContentType, contentTypeJSON)

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
