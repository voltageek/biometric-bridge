package tray

// Updater stub for MVP - no network.

import "strings"

// CheckForUpdates returns a user-facing string describing update status.
// For MVP this is a no-network stub that displays current version or a
// not-configured message.
func CheckForUpdates(currentVersion string, configuredURL string) string {
	if strings.TrimSpace(configuredURL) == "" {
		if strings.TrimSpace(currentVersion) == "" {
			return "Update checking not configured"
		}
		return "Version: " + currentVersion
	}
	// Network-based checking is post-MVP; for now show current version
	return "Version: " + currentVersion
}

// Version is the build-time application version. It may be set via ldflags.
var Version = ""
