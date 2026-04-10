<!--
  SYNC IMPACT REPORT
  ==================
  Version change: 0.0.0 (unfilled template) -> 1.0.0
  Bump rationale: MAJOR — initial constitution ratification (all
  principles, sections, and governance newly established).

  Modified principles:
    - [PRINCIPLE_1_NAME] -> I. Localhost-Only Security
    - [PRINCIPLE_2_NAME] -> II. Driver Abstraction
    - [PRINCIPLE_3_NAME] -> III. Zero Biometric Storage
    - [PRINCIPLE_4_NAME] -> IV. JWT-Only Authentication
    - [PRINCIPLE_5_NAME] -> V. Resilient Reconnection
    - (added)            -> VI. SDK-Agnostic API Surface
    - (added)            -> VII. Fail-Fast Startup

  Added sections:
    - Security Constraints (was [SECTION_2_NAME])
    - Development Workflow (was [SECTION_3_NAME])
    - Governance (filled from [GOVERNANCE_RULES])

  Removed sections: none

  Templates requiring updates:
    - .specify/templates/plan-template.md       ✅ reviewed (no changes needed;
      Constitution Check section is generic and will be filled per-feature)
    - .specify/templates/spec-template.md       ✅ reviewed (no changes needed;
      requirements section is generic)
    - .specify/templates/tasks-template.md      ✅ reviewed (no changes needed;
      phase structure is generic)
    - .specify/templates/commands/*.md           ✅ no command files exist

  Follow-up TODOs:
    - RATIFICATION_DATE set to 2026-04-10 (today, first ratification)
-->

# Biometric Bridge Constitution

## Core Principles

### I. Localhost-Only Security

The bridge process MUST bind exclusively to `127.0.0.1`. It MUST NOT
be reachable from the network under any default configuration. The
loopback binding is the first line of defense: biometric hardware is
never exposed to the internet, even indirectly.

- The listen address MUST default to `127.0.0.1:7070`.
- Configuration MUST NOT provide a convenience toggle to bind to
  `0.0.0.0` or any non-loopback address.
- CORS `Access-Control-Allow-Origin` MUST be set to a single
  configured origin matching the web application's domain. Wildcard
  origins (`*`) are forbidden.

**Rationale**: The bridge handles raw biometric templates. Network
exposure — even behind a firewall — creates an unnecessary attack
surface for data that never needs to leave the workstation.

### II. Driver Abstraction

All SDK functionality MUST be accessed through the single `Driver`
interface defined in `internal/driver/driver.go`. No package outside
`internal/driver/` MUST import an SDK-specific package directly.

- New SDK integrations MUST implement the full `Driver` interface
  (`Connect`, `Devices`, `Enroll`, `Scan`, `Subscribe`, `Close`).
- Driver selection MUST be resolved at build time (build tags) or
  config time (`driver:` key), never at runtime via reflection or
  dynamic loading.
- The HTTP/WebSocket layer MUST remain completely SDK-agnostic.

**Rationale**: The bridge supports two fundamentally different SDKs
(G-SDK via gRPC, BS2 via CGo). A clean interface boundary prevents
SDK concerns from leaking into transport, auth, or event logic.

### III. Zero Biometric Storage

The bridge MUST NOT persist, cache, or log biometric template data
at rest. Templates flow through the bridge as transient in-memory
values and are returned to the caller immediately.

- Template bytes MUST NOT be written to disk, database, or any
  durable store by the bridge process.
- Log output MUST NOT include template bytes, even at debug level.
- The bridge has no knowledge of previously enrolled templates;
  all matching is performed server-side.

**Rationale**: Centralising biometric storage on the server reduces
the compliance surface (GDPR, BIPA) and ensures a single point of
audit and deletion.

### IV. JWT-Only Authentication

Every authenticated endpoint MUST require a valid JWT. The bridge
MUST validate the token signature, issuer (`iss`), audience (`aud`),
and expiry (`exp`) on every request. No alternative authentication
mechanism (API keys, basic auth, cookies) is permitted.

- Algorithm MUST be `ES256` (ECDSA P-256).
- The bridge MUST hold only the public key; the private signing key
  MUST remain on the backend server.
- Tokens without an `exp` claim MUST be rejected.
- The `aud` claim MUST equal `"biometric-bridge"`.
- WebSocket connections MUST receive the JWT via the `?token=` query
  parameter (browsers cannot set custom headers on WS upgrades).
- The frontend MUST hold the JWT in memory only — not
  `localStorage` or `sessionStorage`.

**Rationale**: Short-lived, scoped JWTs limit the blast radius if a
token is extracted from browser memory. Public-key-only validation
means a compromised bridge cannot mint tokens.

### V. Resilient Reconnection

Drivers MUST implement exponential backoff reconnection internally.
The WebSocket connection between the browser and the bridge MUST
remain open during driver-level reconnection — clients MUST NOT be
disconnected due to transient SDK or device failures.

- Backoff MUST start at the configured base (default 1 s) and cap
  at the configured maximum (default 120 s).
- During reconnection, the driver MUST emit `reconnecting` events
  (with attempt count and wait duration) so the frontend can show
  status to the user.
- On successful reconnection, the driver MUST emit a `connected`
  event and reset the backoff counter.
- Each device MUST reconnect independently of other devices.

**Rationale**: Fingerprint readers are physical devices on
potentially unreliable USB or LAN links. Silent reconnection with
user-visible status events prevents workflow interruption.

### VI. SDK-Agnostic API Surface

The HTTP and WebSocket API (routes, request/response shapes, event
types) MUST be identical regardless of which driver is active. A
frontend developer MUST NOT need to know or care whether the G-SDK
or BS2 driver is in use.

- The route table (`/healthz`, `/api/devices`, `/api/enroll`,
  `/api/scan`, `/events`) MUST NOT change between drivers.
- JSON field names and types MUST be stable across drivers.
- Driver-specific information (e.g., internal SDK error codes) MUST
  be mapped to the standard error response format (400/401/502/503)
  before reaching the client.

**Rationale**: The web frontend is developed and tested once. If the
API surface varies by driver, every frontend change requires
multi-driver regression testing.

### VII. Fail-Fast Startup

The bridge MUST connect to every configured device during startup.
If any device fails to connect, the process MUST exit with a
non-zero status code and a descriptive log line — the HTTP server
MUST NOT begin accepting requests.

- Partial startup (some devices connected, some not) is forbidden.
- The process MUST exit cleanly so service managers (systemd,
  launchd) can apply their restart policies.
- Startup failures MUST produce a single, human-readable log line
  identifying the failing device and the error.

**Rationale**: A half-connected bridge creates confusing partial
failures at scan time. Fail-fast ensures problems are caught during
deployment, not during a user's enrollment session.

## Security Constraints

These constraints supplement the principles above and apply across
the entire codebase.

- **Network binding**: `127.0.0.1` only. See Principle I.
- **Token scope**: Every JWT MUST carry `iss`, `exp`, `sub`, and
  `aud` claims. The bridge MUST validate `iss`, `exp`, and `aud`.
  Recommended TTL is 15 minutes.
- **Clock skew tolerance**: Configurable (default 30 s) but MUST
  NOT exceed 60 s.
- **CORS**: Single-origin only. The JWT is the sole guard for
  WebSocket connections (CORS does not protect WS).
- **No secret material on workstation**: The bridge holds only the
  ECDSA public key. Private keys, user databases, and biometric
  stores reside on the backend server.
- **TLS for G-SDK**: Communication between the bridge and the
  Suprema Device Gateway MUST use TLS 1.2 with a CA certificate
  pinned via configuration (`gateway_ca_cert`).

## Development Workflow

- **Language**: Go. Pure Go for the G-SDK driver; CGo required for
  the BS2 driver only.
- **Build tags**: `gsdk` or `bs2` — exactly one MUST be active per
  build. Builds that accidentally include both MUST fail.
- **Driver preference**: G-SDK (pure Go, cross-compilable) is the
  default unless the deployment environment lacks the Suprema Device
  Gateway binary.
- **Project layout**: The canonical layout in
  `gsdk-bridge-spec.md` section 12 is authoritative. New packages
  MUST be placed under `internal/` unless they are intended for
  external consumption (none currently).
- **Testing discipline**: Every `Driver` interface method MUST have
  at least one integration test against a mock or real device.
  Auth middleware MUST have unit tests covering valid, expired,
  wrong-audience, and missing-token cases.
- **Configuration**: `config.yaml` next to the binary. Only the
  matching driver block is read. Unknown keys MUST produce a
  startup warning, not a silent ignore.

## Governance

This constitution is the highest-authority document for the
Biometric Bridge project. It supersedes informal practices,
PR-level conventions, and ad-hoc decisions.

### Amendment procedure

1. Propose the change as a PR modifying this file.
2. The PR description MUST state which principle(s) are affected
   and classify the change as MAJOR, MINOR, or PATCH per the
   versioning policy below.
3. At least one maintainer MUST review and approve.
4. After merge, propagate changes to dependent templates
   (plan-template, spec-template, tasks-template) if the
   amendment alters mandatory sections or constraints.

### Versioning policy

- **MAJOR**: Removal or backward-incompatible redefinition of a
  principle. Requires migration notes.
- **MINOR**: Addition of a new principle or material expansion of
  existing guidance.
- **PATCH**: Clarifications, wording fixes, typo corrections,
  non-semantic refinements.

### Compliance review

- Every PR MUST include a self-check against the 7 principles.
  The plan-template Constitution Check section provides the gate.
- Violations MUST be documented in the Complexity Tracking table
  of the implementation plan with justification and rejected
  alternatives.

**Version**: 1.0.0 | **Ratified**: 2026-04-10 | **Last Amended**: 2026-04-10
