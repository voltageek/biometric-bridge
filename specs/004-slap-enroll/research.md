# Research: Slap Enrollment (Multi-Finger)

**Feature**: 004-slap-enroll | **Date**: 2026-04-16 | **Plan**: [plan.md](./plan.md)

## Overview

This document captures research findings and design decisions for the slap enrollment feature. No NEEDS CLARIFICATION items were identified in the Technical Context - all technologies and patterns are already established in the codebase.

---

## Research 1: Quality Validation Placement

**Question**: Where should per-finger quality validation occur - in the driver or API layer?

**Decision**: Quality validation at API layer (handler)

**Rationale**:
- Quality thresholds are business logic, not hardware/SDK concerns
- Drivers should remain simple adapters between SDK and internal types
- Different endpoints may want different quality behaviors (scan vs enroll)
- Allows configurable thresholds without driver changes

**Alternatives Considered**:
1. **Driver-level validation**: Rejected because it couples business rules to SDK integration and would require passing config to drivers
2. **Middleware validation**: Rejected because quality depends on capture result, not request

---

## Research 2: Retry Granularity

**Question**: Should retries be per-finger or per-impression?

**Decision**: Per-impression retry (whole slap recaptured)

**Rationale**:
- Matches hardware behavior - user lifts and replaces all fingers
- Cannot selectively re-scan individual fingers in slap mode
- Simpler UX messaging ("place fingers again" vs tracking individual failures)
- Quality often improves across all fingers on repositioning

**Alternatives Considered**:
1. **Per-finger retry with combining**: Rejected because hardware doesn't support selective capture and combining images from different placements has matching issues

---

## Research 3: Event Emission from API Handlers

**Question**: How should API handlers emit WebSocket events (e.g., `enrollment_retry`)?

**Decision**: Add `Emit()` method to events.Broker for direct event injection

**Rationale**:
- Current broker only reads from driver's Subscribe channel
- Retry events are API-layer concerns, not driver events
- Direct emit avoids creating a fake driver channel
- Maintains separation: drivers emit hardware events, handlers emit workflow events

**Alternatives Considered**:
1. **Create second channel for handler events**: Rejected because it complicates broker with multiple sources
2. **Have handlers write to driver's channel**: Rejected because it violates driver abstraction
3. **Don't emit retry events**: Rejected because spec requires WebSocket notifications (FR-005)

**Implementation**:
```go
// Add to events/broker.go
func (b *Broker) Emit(evt driver.Event) {
    b.fanOut(evt)
}
```

---

## Research 4: Strict Mode Default

**Question**: Should strict mode (reject on missing fingers) be default on or off?

**Decision**: `strict: true` as default

**Rationale**:
- Most enrollments expect all fingers present
- Missing fingers usually indicates placement error, not intentional
- Lenient mode is opt-in for users with missing/injured fingers
- Matches NIST enrollment guidelines for expected finger count

**Alternatives Considered**:
1. **Default lenient**: Rejected because it would silently accept incomplete enrollments

---

## Research 5: Timeout Configuration

**Question**: Should timeouts be configurable via request or fixed?

**Decision**: Fixed timeouts (60s overall, 20s per-capture)

**Rationale**:
- Timeouts are safety bounds, not user preferences
- Per-request timeout would complicate handler logic
- Config file override could be added later if needed
- 60s allows 2 captures + 3 retries with margin

**Alternatives Considered**:
1. **Request-level timeout parameter**: Rejected to minimize API surface; can add later if needed
2. **Config file only**: Considered acceptable but not implementing for v1

---

## Research 6: Error Response Structure

**Question**: What information should quality failure errors include?

**Decision**: Include failed fingers with their quality scores and threshold

**Rationale**:
- Helps operators guide users on which fingers need better placement
- Useful for debugging and audit logging
- Matches existing error patterns in codebase

**Response Structure**:
```json
{
  "error": "enrollment failed: quality threshold not met after 3 retries",
  "failedFingers": [
    {"finger": "right_index", "quality": 45, "threshold": 60}
  ]
}
```

---

## Research 7: Existing SlapScan Interface Sufficiency

**Question**: Does the existing `driver.SlapScan()` method provide all needed functionality?

**Decision**: Yes, existing interface is sufficient

**Rationale**:
- `SlapScan(ctx, deviceName, mode) (*SlapScanResult, error)` returns:
  - Full slap image (bytes, width, height)
  - Segmented fingers with position, template, dimensions, quality
- Quality scores are per-finger NIST scores
- ErrSlapNotSupported already defined for BS2
- No interface changes needed

**Evidence** (from `internal/driver/driver.go`):
```go
type SlapScanResult struct {
    SlapImage  []byte       // Full slap image
    SlapWidth  int
    SlapHeight int
    Fingers    []ScanResult // Segmented fingers with Quality field
}
```

---

## Best Practices Applied

### Go HTTP Handler Patterns
- Use closure-based handler factory (`NewSlapEnrollHandler(deps) http.HandlerFunc`)
- Define request/response types as private structs
- Use helper functions for JSON error responses
- Context propagation for timeouts and cancellation

### Testing Strategy
- Table-driven tests for validation edge cases
- Mock driver for integration tests
- Demo driver for end-to-end simulation
- Hardware testing as final validation

### Error Handling
- Map SDK errors to HTTP status codes consistently
- Include actionable information in error messages
- Log with sufficient context for debugging

---

## Unresolved Items

None. All technical decisions are clear and implementation can proceed.
