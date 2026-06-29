// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package protocolreg

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// ArtifactReport describes the state of protocol-registration build artefacts
// on the current platform. An "artefact" in this context means any file or
// registry entry written during registration (desktop file, wrapper script,
// app bundle, registry key).
type ArtifactReport struct {
	// Platform is the runtime.GOOS value.
	Platform string `json:"platform"`
	// Artifacts lists the expected artefact paths/keys for this platform.
	Artifacts []ArtifactEntry `json:"artifacts"`
	// AllPresent is true when every expected artefact exists.
	AllPresent bool `json:"all_present"`
}

// ArtifactEntry describes a single registration artefact.
type ArtifactEntry struct {
	// Name is a human-readable label (e.g. "desktop file", "wrapper script").
	Name string `json:"name"`
	// Path is the filesystem path or registry key for this artefact.
	Path string `json:"path"`
	// Present is true when the artefact was found on disk / in the registry.
	Present bool `json:"present"`
	// Valid is true when the artefact exists AND its content is structurally correct
	// (e.g. references the right executable, declares the right MIME type).
	Valid bool `json:"valid"`
	// Issue describes what is wrong when Valid is false and Present is true.
	Issue string `json:"issue,omitempty"`
	// Hint is an actionable remediation step when Present or Valid is false.
	Hint string `json:"hint,omitempty"`
}

// DiscoverArtifacts enumerates the expected registration artefacts for the
// current platform and reports which are present and structurally valid.
// It performs read-only filesystem (or registry) inspection — it never
// modifies system state.
//
// This is the primary entry point for issue #357: "Improve protocol
// registration for build artifact discovery."
func (r *Registrar) DiscoverArtifacts() *ArtifactReport {
	report := &ArtifactReport{Platform: runtime.GOOS}

	switch runtime.GOOS {
	case "linux":
		r.discoverArtifactsLinux(report)
	case "darwin":
		r.discoverArtifactsDarwin(report)
	case "windows":
		r.discoverArtifactsWindows(report)
	default:
		// Unsupported platform: produce a single informational entry.
		report.Artifacts = append(report.Artifacts, ArtifactEntry{
			Name:  "platform_support",
			Path:  "",
			Issue: fmt.Sprintf("artifact discovery is not supported on %s", runtime.GOOS),
			Hint:  "Protocol registration is only supported on Linux, macOS, and Windows.",
		})
		return report
	}

	report.AllPresent = allArtifactsPresent(report.Artifacts)
	return report
}

// discoverArtifactsLinux checks the two artefacts written by registerLinux:
// the .desktop file and the wrapper shell script.
func (r *Registrar) discoverArtifactsLinux(report *ArtifactReport) {
	// ── desktop file ─────────────────────────────────────────────────────────
	desktopEntry := ArtifactEntry{
		Name: "desktop file",
		Path: r.linuxDesktopPath(),
	}
	desktopBytes, err := os.ReadFile(r.linuxDesktopPath())
	if err != nil {
		desktopEntry.Present = false
		desktopEntry.Issue = fmt.Sprintf("file not found: %v", err)
		desktopEntry.Hint = "Run 'glassbox protocol:register' to create the desktop file, " +
			"or manually place it at " + r.linuxDesktopPath()
	} else {
		desktopEntry.Present = true
		issues := validateDesktopFileContent(string(desktopBytes), r)
		if len(issues) == 0 {
			desktopEntry.Valid = true
		} else {
			desktopEntry.Issue = strings.Join(issues, "; ")
			desktopEntry.Hint = "Run 'glassbox protocol:repair' to rewrite the desktop file with correct content."
		}
	}
	report.Artifacts = append(report.Artifacts, desktopEntry)

	// ── wrapper script ────────────────────────────────────────────────────────
	wrapperEntry := ArtifactEntry{
		Name: "wrapper script",
		Path: r.linuxWrapperPath(),
	}
	wrapperBytes, err := os.ReadFile(r.linuxWrapperPath())
	if err != nil {
		wrapperEntry.Present = false
		wrapperEntry.Issue = fmt.Sprintf("file not found: %v", err)
		wrapperEntry.Hint = "Run 'glassbox protocol:repair' to recreate the wrapper script at " + r.linuxWrapperPath()
	} else {
		wrapperEntry.Present = true
		if strings.Contains(string(wrapperBytes), r.executablePath) {
			wrapperEntry.Valid = true
		} else {
			extracted := extractExecPath(string(wrapperBytes))
			if extracted == "" {
				wrapperEntry.Issue = "wrapper script does not contain a recognisable exec line"
			} else {
				wrapperEntry.Issue = fmt.Sprintf(
					"wrapper script references %q instead of %q — registration is stale",
					extracted, r.executablePath)
			}
			wrapperEntry.Hint = "Run 'glassbox protocol:repair' to update the wrapper script."
		}
	}
	report.Artifacts = append(report.Artifacts, wrapperEntry)
}

// discoverArtifactsDarwin checks the two artefacts written by registerDarwin:
// the Info.plist and the app bundle executable.
func (r *Registrar) discoverArtifactsDarwin(report *ArtifactReport) {
	// ── Info.plist ────────────────────────────────────────────────────────────
	plistEntry := ArtifactEntry{
		Name: "Info.plist",
		Path: r.macOSPlistPath(),
	}
	plistBytes, err := os.ReadFile(r.macOSPlistPath())
	if err != nil {
		plistEntry.Present = false
		plistEntry.Issue = fmt.Sprintf("file not found: %v", err)
		plistEntry.Hint = "Run 'glassbox protocol:register' to create the macOS app bundle."
	} else {
		plistEntry.Present = true
		if strings.Contains(string(plistBytes), "<key>CFBundleURLSchemes</key>") &&
			strings.Contains(string(plistBytes), "<string>"+Scheme+"</string>") {
			plistEntry.Valid = true
		} else {
			plistEntry.Issue = "Info.plist does not declare the glassbox:// URL scheme"
			plistEntry.Hint = "Run 'glassbox protocol:repair' to rewrite the Info.plist."
		}
	}
	report.Artifacts = append(report.Artifacts, plistEntry)

	// ── app bundle executable ─────────────────────────────────────────────────
	execEntry := ArtifactEntry{
		Name: "app bundle executable",
		Path: r.macOSExecutablePath(),
	}
	execBytes, err := os.ReadFile(r.macOSExecutablePath())
	if err != nil {
		execEntry.Present = false
		execEntry.Issue = fmt.Sprintf("file not found: %v", err)
		execEntry.Hint = "Run 'glassbox protocol:repair' to recreate the app bundle executable."
	} else {
		execEntry.Present = true
		if strings.Contains(string(execBytes), r.executablePath) {
			execEntry.Valid = true
		} else {
			extracted := extractExecPath(string(execBytes))
			if extracted == "" {
				execEntry.Issue = "app bundle executable does not contain a recognisable exec line"
			} else {
				execEntry.Issue = fmt.Sprintf(
					"app bundle executable references %q instead of %q — registration is stale",
					extracted, r.executablePath)
			}
			execEntry.Hint = "Run 'glassbox protocol:repair' to update the app bundle executable."
		}
	}
	report.Artifacts = append(report.Artifacts, execEntry)
}

// discoverArtifactsWindows checks the registry artefacts written by registerWindows.
func (r *Registrar) discoverArtifactsWindows(report *ArtifactReport) {
	// ── root registry key ─────────────────────────────────────────────────────
	rootKey := ArtifactEntry{
		Name: "root registry key",
		Path: windowsRegistryKey,
	}
	regOutput, err := runCommand("reg", "query", windowsRegistryKey)
	if err != nil {
		rootKey.Present = false
		rootKey.Issue = fmt.Sprintf("registry key not found: %v", err)
		rootKey.Hint = "Run 'glassbox protocol:register' to create the registry entries. " +
			"Administrator privileges may be required."
		report.Artifacts = append(report.Artifacts, rootKey)
		return
	}
	rootKey.Present = true
	if strings.Contains(regOutput, "URL Protocol") {
		rootKey.Valid = true
	} else {
		rootKey.Issue = "registry key is missing the 'URL Protocol' value"
		rootKey.Hint = "Run 'glassbox protocol:repair' to restore the missing registry value."
	}
	report.Artifacts = append(report.Artifacts, rootKey)

	// ── shell open command ────────────────────────────────────────────────────
	openCmd := ArtifactEntry{
		Name: "shell open command",
		Path: windowsRegistryKey + `\shell\open\command`,
	}
	cmdOutput, err := runCommand("reg", "query", windowsRegistryKey+`\shell\open\command`, "/ve")
	if err != nil {
		openCmd.Present = false
		openCmd.Issue = fmt.Sprintf("shell open command registry key not found: %v", err)
		openCmd.Hint = "Run 'glassbox protocol:repair' to add the shell open command."
	} else {
		openCmd.Present = true
		if strings.Contains(cmdOutput, r.executablePath) {
			openCmd.Valid = true
		} else {
			openCmd.Issue = fmt.Sprintf(
				"shell open command does not reference the current binary %q — registration is stale",
				r.executablePath)
			openCmd.Hint = "Run 'glassbox protocol:repair' to update the shell open command."
		}
	}
	report.Artifacts = append(report.Artifacts, openCmd)
}

// validateDesktopFileContent returns a list of content-level issues for a Linux
// .desktop file. An empty list means the content is valid.
func validateDesktopFileContent(content string, r *Registrar) []string {
	var issues []string
	if !strings.Contains(content, "MimeType="+linuxMimeType+";") {
		issues = append(issues, fmt.Sprintf("missing MimeType=%s; declaration", linuxMimeType))
	}
	if !strings.Contains(content, "Exec="+r.linuxWrapperPath()) {
		issues = append(issues, fmt.Sprintf(
			"Exec entry does not reference the wrapper at %s", r.linuxWrapperPath()))
	}
	return issues
}

// allArtifactsPresent returns true when every entry in the slice is Present.
func allArtifactsPresent(entries []ArtifactEntry) bool {
	for _, e := range entries {
		if !e.Present {
			return false
		}
	}
	return len(entries) > 0
}
