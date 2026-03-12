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
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
)

// KerberosManager handles Kerberos authentication.
type KerberosManager struct{}

// NewKerberosManager creates a new KerberosManager.
func NewKerberosManager() *KerberosManager {
	return &KerberosManager{}
}

// Authenticate performs Kerberos authentication using kinit.
// Returns additional JAVA_OPTS for Oracle Kerberos authentication if successful.
func (m *KerberosManager) Authenticate(args KerberosArgs) (javaOpts string, cleanup func(), err error) {
	if args.UserPrincipal == "" {
		// Kerberos not configured
		return "", nil, nil
	}

	logrus.Infof("Initiating Kerberos authentication for principal: %s", args.UserPrincipal)

	if args.Password != "" {
		// Password-based authentication
		err = m.authenticateWithPassword(args.UserPrincipal, args.Password)
	} else if args.KeytabFilePath != "" {
		// Keytab-based authentication
		err = m.authenticateWithKeytab(args.UserPrincipal, args.KeytabFilePath)
	} else {
		return "", nil, fmt.Errorf("PLUGIN_KERBEROS_USER_PRINCIPAL requires an authentication method. Set either PLUGIN_KERBEROS_PASSWORD or PLUGIN_KERBEROS_KEYTAB_FILE_PATH")
	}

	if err != nil {
		return "", nil, err
	}

	logrus.Infof("Kerberos authentication successful for principal: %s", args.UserPrincipal)

	// Return cleanup function to destroy Kerberos ticket
	cleanup = func() {
		if _, err := runCommand("kdestroy"); err != nil {
			logrus.Warnf("Failed to destroy Kerberos ticket: %v", err)
		}
	}

	// Return JAVA_OPTS for Oracle Kerberos authentication
	return "-Doracle.net.authentication_services=KERBEROS5", cleanup, nil
}

// authenticateWithPassword performs kinit with password.
func (m *KerberosManager) authenticateWithPassword(principal, password string) error {
	cmd := exec.Command("kinit", "-f", principal)
	cmd.Stdin = strings.NewReader(password)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("password-based Kerberos authentication kinit failed: %w", err)
	}

	return nil
}

// authenticateWithKeytab performs kinit with keytab file.
func (m *KerberosManager) authenticateWithKeytab(principal, keytabPath string) error {
	if !fileExists(keytabPath) {
		return fmt.Errorf("keytab file not found: %s", keytabPath)
	}

	_, err := runCommand("kinit", "-f", "-k", "-t", keytabPath, principal)
	if err != nil {
		return fmt.Errorf("keytab-based Kerberos authentication kinit failed: %w", err)
	}

	return nil
}
