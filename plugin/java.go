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
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// detectJavaHome finds the Java installation directory and exports it.
func detectJavaHome() string {
	if javaHome := os.Getenv("JAVA_HOME"); javaHome != "" {
		logrus.Debugf("Using JAVA_HOME from environment: %s", javaHome)
		return javaHome
	}

	// Primary method: parse java.home from JVM settings
	if javaHome := detectJavaHomeFromSettings(); javaHome != "" {
		logrus.Debugf("Detected JAVA_HOME from java settings: %s", javaHome)
		setJavaHomeEnv(javaHome)
		return javaHome
	}

	// Fallback: common Java installation paths
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

	logrus.Warn("Could not detect JAVA_HOME")
	return ""
}

// detectJavaHomeFromSettings uses java -XshowSettings:properties to find java.home
func detectJavaHomeFromSettings() string {
	output, err := runCommand("java", "-XshowSettings:properties", "-version")
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "java.home") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

// setJavaHomeEnv exports JAVA_HOME to environment.
func setJavaHomeEnv(javaHome string) {
	if javaHome != "" {
		os.Setenv("JAVA_HOME", javaHome)
	}
}
