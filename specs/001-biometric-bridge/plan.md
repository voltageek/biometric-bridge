# Implementation Plan: Biometric Bridge

**Branch**: `001-biometric-bridge` | **Date**: 2026-04-10 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-biometric-bridge/spec.md`

## Summary

Build a lightweight Go binary that binds to `127.0.0.1:7070` and provides a web application's frontend with authenticated HTTP and WebSocket access to Suprema fingerprint readers. The bridge abstracts two SDK backends (G-SDK via gRPC, BioStar 2 Device SDK via CGo) behind a single `Driver` interface, exposing scan, enroll, device listing, and real-time event streaming through a unified, SDK-agnostic API surface. All requests (except `/healthz`) require a short-lived ES256 JWT issued by the remote backend server. The bridge stores no biometric data.

## Technical Context

**Language/Version**: Go 1.22+
**Primary Dependencies**: BS2 Device SDK via CGo (primary driver; G-SDK deferred pending license key), `github.com/golang-jwt/jwt/v5` (JWT validation), `github.com/gorilla/websocket` (WebSocket), `gopkg.in/yaml.v3` (config), `log/slog` (structured logging)
**Storage**: N/A (zero persistence by constitution)
**Testing**: `go test`, `testing` stdlib; integration tests against mock driver
**Target Platform**: Linux amd64 (primary), Windows amd64, macOS amd64/arm64
**Project Type**: Local daemon / system service
**Performance Goals**: Scan response < 5s after finger placement; event delivery < 2s; health check < 500ms; startup < 15s
**Constraints**: Localhost-only binding; single active driver per build; max 10 concurrent WebSocket subscribers; 10s scan/enroll timeout
**Scale/Scope**: 1-10 concurrent fingerprint readers; 1-10 WebSocket subscribers; single-workstation deployment

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| # | Principle | Status | Notes |
|---|-----------|--------|-------|
| I | Localhost-Only Security | PASS | Binds to `127.0.0.1:7070`; no toggle for `0.0.0.0`; single-origin CORS |
| II | Driver Abstraction | PASS | Single `Driver` interface in `internal/driver/driver.go`; no SDK imports outside `internal/driver/` |
| III | Zero Biometric Storage | PASS | Templates returned immediately; never written to disk or log |
| IV | JWT-Only Authentication | PASS | ES256, public-key-only validation; `iss`/`aud`/`exp` enforced; no alternative auth |
| V | Resilient Reconnection | PASS | Exponential backoff in each driver; WS stays open during reconnect; status events emitted |
| VI | SDK-Agnostic API Surface | PASS | Identical routes, JSON shapes, event types regardless of active driver |
| VII | Fail-Fast Startup | PASS | All devices must connect before HTTP server accepts; non-zero exit on failure |

No violations. Complexity Tracking table not needed.

## Project Structure

### Documentation (this feature)

```text
specs/001-biometric-bridge/
├── plan.md              # This file
├── research.md          # Phase 0: SDK research and spike findings
├── data-model.md        # Phase 1: entities, enums, state transitions
├── quickstart.md        # Phase 1: local dev setup and run instructions
├── contracts/           # Phase 1: API contracts (HTTP + WebSocket)
│   ├── http-api.md      # REST endpoint contracts
│   └── websocket-api.md # WebSocket event contracts
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
biometric-bridge/
├── cmd/
│   └── bridge/
│       └── main.go              # Entry point: config load, driver init, server start
├── internal/
│   ├── config/
│   │   └── config.go            # YAML config parsing and validation (FR-018)
│   ├── driver/
│   │   ├── driver.go            # Driver interface + shared types (DeviceConfig, DeviceInfo, etc.)
│   │   ├── gsdk/
│   │   │   └── driver.go        # G-SDK implementation (build tag: gsdk)
│   │   └── bs2/
│   │       └── driver.go        # BS2 implementation (build tag: bs2)
│   ├── auth/
│   │   ├── keys.go              # ECDSA public key loading from PEM file
│   │   ├── token.go             # JWT validation (ES256, iss, aud, exp)
│   │   └── middleware.go        # HTTP middleware + WS token extraction
│   ├── api/
│   │   ├── router.go            # Route registration, CORS, health check
│   │   ├── devices.go           # GET /api/devices handler
│   │   ├── enroll.go            # POST /api/enroll handler
│   │   └── scan.go              # POST /api/scan handler
│   ├── events/
│   │   ├── broker.go            # Fan-out hub: driver events → WebSocket subscribers
│   │   └── handler.go           # GET /events WebSocket upgrade + token expiry enforcement
│   └── device/
│       └── registry.go          # Per-device busy lock (FR-015) + name→ID mapping
├── install/
│   ├── biometric-bridge.service # systemd unit file
│   └── biometric-bridge.plist   # launchd plist file
├── config.yaml                  # Default config (checked in as example)
├── go.mod
└── go.sum
```

**Structure Decision**: Single Go binary project following the canonical layout from the bridge spec (section 12). The `internal/` convention enforces encapsulation. The `events/` package is separated from `api/` because the WebSocket event broker has its own lifecycle (runs for the duration of the process, not per-request). The `device/` package provides the per-device mutex registry that both `api/` and `events/` depend on. No additional projects or services are needed.

## Complexity Tracking

> No violations. Table intentionally empty.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| — | — | — |
