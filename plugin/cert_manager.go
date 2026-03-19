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
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	trustStoreFileName    = "cacerts"
	keyStoreFileName      = "jssecacerts"
	clientPKCS12FileName  = "client.p12"
	rootCAAliasName       = "mongodb-root-ca-cert"
	clientCertAliasName   = "client_pkcs12"
)

// CertManager handles SSL/TLS certificate configuration for Java.
type CertManager struct {
	certsDir      string
	storePassword string
	javaHome      string
}

// NewCertManager creates a new CertManager.
func NewCertManager(args CertArgs, javaHome string) *CertManager {
	return &CertManager{
		certsDir:      args.CertsDir,
		storePassword: args.StorePassword,
		javaHome:      javaHome,
	}
}

// detectJavaHome finds the Java installation directory.
func detectJavaHome() string {
	// Check if JAVA_HOME is already set in environment
	if javaHome := os.Getenv("JAVA_HOME"); javaHome != "" {
		logrus.Debugf("Using JAVA_HOME from environment: %s", javaHome)
		setJavaHomeEnv(javaHome)
		return javaHome
	}

	// Primary method: parse java.home from JVM settings
	if javaHome := detectJavaHomeFromSettings(); javaHome != "" {
		logrus.Debugf("Detected JAVA_HOME from java settings: %s", javaHome)
		setJavaHomeEnv(javaHome)
		return javaHome
	}

	// Fallback: Common Java installation paths
	commonPaths := []string{
		"/opt/java/openjdk",
		"/usr/lib/jvm/default-jvm",
		"/usr/lib/jvm/java-21-openjdk",
		"/usr/lib/jvm/java-17-openjdk",
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(filepath.Join(path, "bin", "java")); err == nil {
			logrus.Debugf("Found JAVA_HOME at common path: %s", path)
			setJavaHomeEnv(path)
			return path
		}
	}

	// Last resort: derive from java binary path
	javaPath, err := runCommand("which", "java")
	if err == nil {
		resolved, err := filepath.EvalSymlinks(strings.TrimSpace(string(javaPath)))
		if err == nil {
			javaHome := filepath.Dir(filepath.Dir(resolved))
			logrus.Debugf("Derived JAVA_HOME from which: %s", javaHome)
			setJavaHomeEnv(javaHome)
			return javaHome
		}
	}

	logrus.Warn("Could not detect JAVA_HOME, certificate import may fail")
	return ""
}

// detectJavaHomeFromSettings uses java -XshowSettings:properties to find java.home
func detectJavaHomeFromSettings() string {
	output, err := runCommand("java", "-XshowSettings:properties", "-version")
	if err != nil {
		return ""
	}

	// Parse output for java.home property
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "java.home") {
			// Format: "java.home = /path/to/java"
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

// setJavaHomeEnv exports JAVA_HOME to environment for Liquibase internals
func setJavaHomeEnv(javaHome string) {
	if javaHome != "" {
		os.Setenv("JAVA_HOME", javaHome)
	}
}

// SetupCertificates configures the Java TrustStore and KeyStore.
// Returns JAVA_OPTS string to be set in environment.
func (m *CertManager) SetupCertificates(args CertArgs) (string, error) {
	var javaOpts []string

	// Ensure certs directory exists with correct permissions
	if err := os.MkdirAll(m.certsDir, 0700); err != nil {
		logrus.Warnf("Failed to create directory %s: %v", m.certsDir, err)
	}
	// Ensure permissions are set even if directory already exists
	if err := os.Chmod(m.certsDir, 0700); err != nil {
		logrus.Warnf("Failed to set permissions on %s: %v", m.certsDir, err)
	}

	// Setup TrustStore if root CA cert exists
	if fileExists(args.SSLCACertPath) {
		trustStoreOpts, err := m.setupTrustStore(args.SSLCACertPath)
		if err != nil {
			logrus.Warnf("TrustStore setup warning: %v", err)
		} else {
			javaOpts = append(javaOpts, trustStoreOpts...)
		}
	}

	// Setup KeyStore if client cert and key exist
	if fileExists(args.ClientCertPath) && fileExists(args.ClientKeyPath) {
		keyStoreOpts, err := m.setupKeyStore(args.ClientCertPath, args.ClientKeyPath)
		if err != nil {
			logrus.Warnf("KeyStore setup warning: %v", err)
		} else {
			javaOpts = append(javaOpts, keyStoreOpts...)
		}
	}

	return strings.Join(javaOpts, " "), nil
}

// setupTrustStore creates and configures the Java TrustStore.
func (m *CertManager) setupTrustStore(caCertPath string) ([]string, error) {
	trustStorePath := filepath.Join(m.certsDir, trustStoreFileName)

	// Copy system truststore if it doesn't exist
	if !fileExists(trustStorePath) {
		systemTrustStore := filepath.Join(m.javaHome, "lib", "security", "cacerts")
		if fileExists(systemTrustStore) {
			if err := copyFile(systemTrustStore, trustStorePath); err != nil {
				logrus.Warnf("Failed to copy system truststore: %v", err)
				logrus.Info("Creating new empty truststore...")
			} else {
				logrus.Infof("Copied system truststore to %s", trustStorePath)
			}
		} else {
			logrus.Warnf("System truststore not found at %s", systemTrustStore)
		}
	} else {
		logrus.Infof("Truststore already exists at %s, skipping copy", trustStorePath)
	}

	// Check if root CA is already imported
	if m.certificateExists(trustStorePath, rootCAAliasName) {
		logrus.Info("Root CA certificate already exists in truststore, skipping import")
	} else {
		logrus.Info("Importing self-signed certificate into trustStore...")
		if err := m.importCertificate(trustStorePath, caCertPath, rootCAAliasName); err != nil {
			return nil, fmt.Errorf("failed to import root CA certificate: %w", err)
		}
		logrus.Infof("Successfully imported root CA certificate into %s", trustStorePath)
	}

	return []string{
		fmt.Sprintf("-Djavax.net.ssl.trustStore=%s", trustStorePath),
		fmt.Sprintf("-Djavax.net.ssl.trustStorePassword=%s", m.storePassword),
	}, nil
}

// setupKeyStore creates and configures the Java KeyStore for client authentication.
func (m *CertManager) setupKeyStore(clientCertPath, clientKeyPath string) ([]string, error) {
	keyStorePath := filepath.Join(m.certsDir, keyStoreFileName)
	pkcs12Path := filepath.Join(m.certsDir, clientPKCS12FileName)

	// Check if client cert is already in keystore
	if fileExists(keyStorePath) && m.certificateExists(keyStorePath, clientCertAliasName) {
		logrus.Infof("Client certificate already exists in keystore at %s, skipping import", keyStorePath)
	} else {
		// Generate PKCS12 from client cert and key
		logrus.Infof("Generating PKCS12 from %s and %s", clientCertPath, clientKeyPath)
		if err := m.createPKCS12(clientCertPath, clientKeyPath, pkcs12Path); err != nil {
			return nil, fmt.Errorf("failed to create PKCS12: %w", err)
		}

		// Import PKCS12 into keystore
		logrus.Info("Importing client certificate into keyStore...")
		if err := m.importPKCS12(keyStorePath, pkcs12Path); err != nil {
			return nil, fmt.Errorf("failed to import client certificate: %w", err)
		}
		logrus.Infof("Successfully imported client certificate into %s", keyStorePath)

		// Clean up PKCS12 file
		if err := os.Remove(pkcs12Path); err != nil {
			logrus.Warnf("Failed to remove temporary PKCS12 file: %v", err)
		}
	}

	return []string{
		fmt.Sprintf("-Djavax.net.ssl.keyStore=%s", keyStorePath),
		fmt.Sprintf("-Djavax.net.ssl.keyStorePassword=%s", m.storePassword),
	}, nil
}

// certificateExists checks if a certificate with the given alias exists in the keystore.
func (m *CertManager) certificateExists(storePath, alias string) bool {
	_, err := runCommand("keytool", "-list",
		"-keystore", storePath,
		"-storepass", m.storePassword,
		"-alias", alias)
	return err == nil
}

// importCertificate imports a certificate into the truststore.
func (m *CertManager) importCertificate(storePath, certPath, alias string) error {
	_, err := runCommand("keytool", "-importcert",
		"-alias", alias,
		"-keystore", storePath,
		"-storepass", m.storePassword,
		"-trustcacerts",
		"-file", certPath,
		"-noprompt")
	return err
}

// createPKCS12 creates a PKCS12 file from certificate and key.
func (m *CertManager) createPKCS12(certPath, keyPath, outputPath string) error {
	_, err := runCommand("openssl", "pkcs12", "-export",
		"-in", certPath,
		"-inkey", keyPath,
		"-out", outputPath,
		"-name", clientCertAliasName,
		"-password", "pass:"+m.storePassword)
	return err
}

// importPKCS12 imports a PKCS12 file into the keystore.
func (m *CertManager) importPKCS12(storePath, pkcs12Path string) error {
	_, err := runCommand("keytool", "-importkeystore",
		"-destkeystore", storePath,
		"-srckeystore", pkcs12Path,
		"-srcstoretype", "PKCS12",
		"-alias", clientCertAliasName,
		"-storepass", m.storePassword,
		"-srcstorepass", m.storePassword,
		"-noprompt")
	return err
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0600)
}
