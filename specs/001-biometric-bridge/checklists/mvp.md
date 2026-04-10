# MVP Requirements Quality Checklist: Biometric Bridge

**Purpose**: Validate that MVP-scoped requirements (Setup, Foundational, US1 Scan, US5 Auth, BS2 driver, Entry Point) are complete, clear, consistent, and implementation-ready
**Created**: 2026-04-10
**Feature**: [spec.md](../spec.md)
**Audience**: Author self-review before implementation
**Depth**: Standard

## Requirement Completeness

- [ ] CHK001 - Are BS2 shared library loading requirements specified, including expected file path, error behavior when library is missing, and `LD_LIBRARY_PATH` expectations? [Completeness, Gap]
- [ ] CHK002 - Is the `config.yaml` validation exhaustive — are all invalid field combinations enumerated (e.g., `driver: bs2` without a `bs2:` section, empty `devices` list, duplicate device names)? [Completeness, Spec §FR-018]
- [ ] CHK003 - Are requirements defined for what happens when `config.yaml` is entirely absent (not just malformed)? [Completeness, Gap]
- [ ] CHK004 - Is the public key file format requirement specified beyond "PEM file" — must it be PKCS8, SEC1, or either? [Completeness, Spec §FR-017]
- [ ] CHK005 - Are requirements for the `clock_skew` config field defined — valid range, default value, behavior when omitted? [Completeness, Gap]
- [ ] CHK006 - Are requirements for `token_issuer` and `token_audience` defaults specified when omitted from config? [Completeness, Gap]
- [ ] CHK007 - Is the scan timeout source of truth clearly defined — is it always 10s hardcoded, or configurable via config? [Completeness, Spec §FR-001]
- [ ] CHK008 - Are CGo build requirements documented — minimum GCC version, `CGO_ENABLED=1` enforcement, build tag mutual exclusivity (`bs2` and `gsdk` cannot both be set)? [Completeness, Gap]

## Requirement Clarity

- [ ] CHK009 - Is "raw template data" in FR-001 defined with enough precision — encoding format (binary vs base64), maximum size, SDK-specific format name? [Clarity, Spec §FR-001]
- [ ] CHK010 - Is the quality score range (0–100) sourced from the SDK or normalized by the bridge? Is this clearly stated? [Clarity, Spec §FR-001]
- [ ] CHK011 - Is "clear error identifying the problem" in FR-017 and FR-018 quantified — must it include file path, field name, expected vs actual value? [Clarity, Spec §FR-017, §FR-018]
- [ ] CHK012 - Is the health check response contract fully specified — just `{"status":"ok"}` or should it include version, uptime, device count? [Clarity, Spec §FR-012]
- [ ] CHK013 - Is "immediately" in FR-015 ("reject ... immediately") defined — does it mean synchronous rejection before any SDK call, or within a time bound? [Clarity, Spec §FR-015]
- [ ] CHK014 - Is the BS2 `lib_path` config field requirement clear — must it be an absolute path, or can it be relative to the binary/working directory? [Clarity, Gap]

## Requirement Consistency

- [ ] CHK015 - Are the JWT claim requirements consistent between the HTTP API contract (http-api.md) and FR-005 in spec.md — do both list the same required claims (`iss`, `aud`, `exp`)? [Consistency, Spec §FR-005, Contracts §http-api]
- [ ] CHK016 - Is the `deviceId` field naming consistent — the API contracts use `deviceId` in JSON but the data model uses `DeviceName` in Go structs; is the mapping requirement explicit? [Consistency, Contracts §http-api, Data Model §1.4]
- [ ] CHK017 - Are error response HTTP status codes consistent between the scan contract (http-api.md) and the spec's edge cases — does the spec's "device busy" always map to 409? [Consistency, Spec §FR-015, Contracts §http-api]
- [ ] CHK018 - Is the startup behavior consistent between FR-011 ("all devices must connect before accepting requests") and the health check (FR-012) — can `/healthz` respond before devices connect, or only after? [Consistency, Spec §FR-011, §FR-012]
- [ ] CHK019 - Are the CORS `allowed_origin` requirements consistent between plan.md ("single configured web application domain") and the config schema — is a trailing slash, port, or wildcard subdomain addressed? [Consistency, Plan §Technical Context]

## Acceptance Criteria Quality

- [ ] CHK020 - Is SC-001 ("within 5 seconds of placing their finger") measurable end-to-end — does it include network latency to the BS2 device, or only SDK call duration? [Measurability, Spec §SC-001]
- [ ] CHK021 - Is SC-005 ("100% of requests without valid authentication tokens are rejected") testable for WebSocket upgrade requests specifically — is the rejection mechanism (HTTP 401 vs WS close frame) specified per scenario? [Measurability, Spec §SC-005]
- [ ] CHK022 - Is SC-008 ("health check within 500ms") measured under what conditions — cold start, steady state, under load? [Measurability, Spec §SC-008]
- [ ] CHK023 - Is SC-011 ("ready within 15 seconds") defined with a clear "ready" signal — is it the first successful HTTP response, a log line, or process exit code? [Measurability, Spec §SC-011]

## Scenario Coverage

- [ ] CHK024 - Are requirements defined for what happens when the BS2 SDK returns an unexpected/undocumented error code during scan? [Coverage, Exception Flow, Gap]
- [ ] CHK025 - Are requirements defined for a scan request arriving while the bridge is still in startup (devices connecting but HTTP not yet listening)? [Coverage, Spec §FR-011]
- [ ] CHK026 - Are requirements defined for concurrent scan requests to *different* devices — should they both proceed in parallel? [Coverage, Spec §FR-015]
- [ ] CHK027 - Are requirements defined for what happens when the BS2 shared library is present but ABI-incompatible (wrong version)? [Coverage, Exception Flow, Gap]
- [ ] CHK028 - Are requirements specified for the bridge's behavior when the system clock is significantly wrong (affecting JWT `exp` validation beyond `clock_skew`)? [Coverage, Edge Case, Gap]
- [ ] CHK029 - Are CORS preflight (OPTIONS) requirements explicitly specified — which paths, required headers, caching? [Coverage, Contracts §http-api]

## Edge Case Coverage

- [ ] CHK030 - Are requirements defined for a scan request with a valid JWT but where the `sub` claim identifies a user without device access — or is authorization out of scope? [Edge Case, Spec §Assumptions]
- [ ] CHK031 - Is the behavior specified when `deviceId` in a scan request is an empty string vs a non-existent device name? Are these different error cases? [Edge Case, Spec §FR-001]
- [ ] CHK032 - Are requirements defined for a JWT that is valid but has additional unexpected claims — should they be ignored or rejected? [Edge Case, Spec §FR-005]
- [ ] CHK033 - Is the behavior specified when the config file has valid YAML but zero devices configured? [Edge Case, Spec §FR-018]
- [ ] CHK034 - Are requirements defined for what happens if the BS2 `BS2_AllocateContext` call fails at startup (SDK initialization failure, not device connection failure)? [Edge Case, Gap]

## Non-Functional Requirements

- [ ] CHK035 - Are structured log output format requirements specified — JSON lines, key names, timestamp format, or left to `slog` defaults? [Clarity, Spec §FR-016]
- [ ] CHK036 - Are log level filtering requirements clear — does "configurable verbosity" mean runtime-changeable or config-time only? [Clarity, Spec §FR-016]
- [ ] CHK037 - Are requirements defined for maximum memory consumption or goroutine limits for the bridge process? [Gap, Non-Functional]
- [ ] CHK038 - Is the listen address requirement restrictive enough — can the user override `127.0.0.1` to `0.0.0.0` via config, violating Constitution Principle I? [Consistency, Spec §FR-006, Plan §Constitution]

## Dependencies & Assumptions

- [ ] CHK039 - Is the assumption that "BS2 Device SDK V2 is forward-compatible across minor versions" documented, or is a pinned version requirement needed? [Assumption, Gap]
- [ ] CHK040 - Is the assumption that fingerprint readers are reachable via TCP/IP documented — or does BS2 also support USB-direct connections that need different requirements? [Assumption, Spec §Assumptions]
- [ ] CHK041 - Are the requirements for the pre-shared public key deployment clear — who generates it, where it must be placed, file permissions? [Dependency, Spec §Assumptions]
- [ ] CHK042 - Is the dependency on `CGO_ENABLED=1` and a C toolchain documented as a deployment prerequisite, not just a build prerequisite? [Dependency, Gap]

## Notes

- Check items off as completed: `[x]`
- Add comments or findings inline
- Items focus on MVP scope: config, auth (US5), scan (US1), BS2 driver, entry point, and foundational infrastructure
- Non-MVP items (enroll, events, device listing, reconnection, G-SDK) are intentionally excluded
- This checklist complements `requirements.md` (general spec quality) with MVP-specific depth
