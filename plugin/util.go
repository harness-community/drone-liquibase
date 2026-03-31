package plugin

import "os"

// applyEnvOverrides sets env vars from the auth overrides map.
func applyEnvOverrides(overrides map[string]string) {
	for key, value := range overrides {
		os.Setenv(key, value)
	}
}

// unsetEnvOverrides removes env vars that were set by auth overrides.
func unsetEnvOverrides(overrides map[string]string) {
	for key := range overrides {
		os.Unsetenv(key)
	}
}
