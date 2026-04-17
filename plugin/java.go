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
	"strconv"
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

// staticJVMFlags are the JVM tuning flags applied to every image variant.
// -XX:+TieredCompilation is the default on modern HotSpot (included for explicitness).
// -XX:TieredStopAtLevel=1 forces C1-only compilation for faster cold starts.
const staticJVMFlags = "-XX:+TieredCompilation -XX:TieredStopAtLevel=1"

// buildJavaOpts assembles the final JAVA_OPTS string from all sources:
//  1. Static JVM tuning flags (always prepended).
//  2. Heap flags computed from cgroup memory limit — skipped if -Xms/-Xmx already present.
//  3. Any extra opts (env JAVA_OPTS, cert opts, kerberos opts) appended in order.
func buildJavaOpts(heapPercent int, extraOpts ...string) string {
	parts := []string{staticJVMFlags}

	for _, opt := range extraOpts {
		if opt != "" {
			parts = append(parts, opt)
		}
	}

	// Skip heap flags if the operator already specified -Xms/-Xmx to avoid silent overrides.
	combined := strings.Join(parts, " ")
	if !strings.Contains(combined, "-Xms") && !strings.Contains(combined, "-Xmx") {
		if heapFlags := computeHeapFlags(heapPercent); heapFlags != "" {
			parts = append(parts, heapFlags)
		}
	} else {
		logrus.Debugf("Skipping heap flags: -Xms/-Xmx already present in JAVA_OPTS")
	}

	return strings.Join(parts, " ")
}

// computeHeapFlags reads the container's cgroup memory limit and returns
// -Xms and -Xmx flags set to heapPercent of the limit.
// Setting Xms=Xmx:
//   - eliminates heap resizing overhead and the GC pauses it triggers,
//   - avoids latency spikes from the OS allocating new memory pages on demand,
//   - gives a predictable memory footprint so the container orchestrator
//     can manage resources accurately and avoid unexpected OOM kills.
//
// Returns empty string if the cgroup limit cannot be read.
func computeHeapFlags(heapPercent int) string {
	return computeHeapFlagsFromPaths(heapPercent,
		"/sys/fs/cgroup/memory.max",
		"/sys/fs/cgroup/memory/memory.limit_in_bytes",
	)
}

// computeHeapFlagsFromPaths reads the container's cgroup memory limit from the
// given paths (tried in order) and returns -Xms/-Xmx flags.
func computeHeapFlagsFromPaths(heapPercent int, cgroupPaths ...string) string {
	var memBytes int64
	var err error
	for _, p := range cgroupPaths {
		memBytes, err = readCgroupMemoryLimit(p)
		if err == nil {
			break
		}
	}
	if err != nil {
		logrus.Warnf("Could not read cgroup memory limit: %v", err)
		return ""
	}

	heapMB := int(memBytes / 1024 / 1024 * int64(heapPercent) / 100)
	if heapMB < 64 {
		logrus.Warnf("Computed heap size %dMB is below minimum threshold (64MB); skipping heap flags", heapMB)
		return ""
	}
	logrus.Debugf("Container memory: %dMB, heap (Xms=Xmx): %dMB (%d%%)", memBytes/1024/1024, heapMB, heapPercent)
	return fmt.Sprintf("-Xms%dm -Xmx%dm", heapMB, heapMB)
}

// readCgroupMemoryLimit reads the memory limit in bytes from a cgroup file.
// Returns an error if the file doesn't exist or contains "max" (unlimited).
func readCgroupMemoryLimit(path string) (int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	value := strings.TrimSpace(string(data))
	if value == "max" || value == "9223372036854771712" {
		return 0, fmt.Errorf("no memory limit set (value: %s)", value)
	}

	memBytes, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse memory limit %q: %w", value, err)
	}

	return memBytes, nil
}
