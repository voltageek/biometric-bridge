# Research — Bridge System Tray GUI

This document records research decisions and resolves open questions from the implementation plan for the Bridge System Tray GUI.

Decisions

- Decision: UI stack
  - Chosen: Fyne for the floating panel + fyne.io/systray for tray integration.
  - Rationale: Fyne provides a portable GUI toolkit for the panel; systray provides reliable cross-platform tray APIs and avoids fighting Fyne for tray semantics. This hybrid was implemented in the codebase.
  - Alternatives: Use a single framework (fyne only) — rejected due to limited tray click/menu guarantees. Electron rejected for size and licensing reasons.

- Decision: Application layout and positioning
  - Chosen: Fixed top-right placement (20px from right, 40px from top) using a best-effort screen-size detector and SetPosition when available.
  - Rationale: Fyne does not expose tray coordinates; a fixed top-right placement meets the user's request with acceptable cross-platform behavior.
  - Alternatives: Attempt to read actual tray icon coordinates (platform-specific). Deferred as complexity for marginal benefit.

- Decision: Single-instance enforcement
  - Chosen: TCP-based instance locker (already present). If second instance detected, it brings first instance to front and exits.
  - Rationale: Simple, cross-platform, avoids duplicate access to hardware.

- Decision: Config & log opening behavior
  - Chosen: Respect BRIDGE_CONFIG env var, fallback to ./config.yaml. "Show Log" attempts a safe LoadDriverOnly and opens the directory containing cfg.Log.File; fallback to current directory.
  - Rationale: Avoid failing UI when config is invalid; allow operator to locate logs quickly.

- Decision: Tray icon
  - Chosen: Generate simple mono icons at multiple sizes (16/24/32/64) programmatically; prefer 64 on macOS and 32 elsewhere. Allow overriding with TRAY_ICON_SIZE env var.
  - Rationale: Provides usable icon that scales on different platforms without requiring external assets during development. Repository contains placeholder base64 assets which can be replaced with designer artwork later.

- Decision: Notifications / toasts
  - Chosen: In-panel transient toasts (temporary label overlay). They appear on Restart/Stop success/failure and hide automatically.
  - Rationale: Simple, cross-platform, no native-notification dependencies.

Open Questions Marked "NEEDS CLARIFICATION" in plan resolved here

- Technical Context: Language/version — Go 1.24.7 (project uses Go 1.24.7 as per repo). Tools: go build, go test.
- Primary dependencies — fyne, fyne.io/systray, golang.design/x/clipboard, github.com/golang-jwt/jwt/v5, gopkg.in/yaml.v3. These are already present in go.mod.

Risks and Mitigations

- Risk: Fyne SetPosition availability varies by version/platform.
  - Mitigation: Use SetPosition only when the method exists; otherwise rely on center/OS placement. The UI remains usable.

- Risk: xrandr/system_profiler commands may be missing on minimal systems.
  - Mitigation: Detector falls back to sensible defaults (1440x900); positioning remains best-effort.

Conclusion

All open research items required to implement the UI and tray behavior have been resolved and implemented in the codebase. The decisions prioritize cross-platform reliability and minimal external dependencies while satisfying the product constraints (localhost-only, no biometric persistence, JWT-only auth).
