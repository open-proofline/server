# Security Policy

Proofline is a private encrypted incident-capture backend. It is not production-ready public infrastructure. The private `/v1` API has no public authentication and must stay behind localhost, WireGuard, a firewall, or an equivalent private boundary.

The current implementation supports generic incident capture and token-scoped read-only incident review. Planned future modes include emergency incidents, non-emergency interaction records, timed safety checks, and evidence notes. Those modes do not change the current vulnerability-reporting process until they are implemented.

## Supported Versions

| Version | Supported |
|---|---|
| 0.5.0 | Yes |
| 0.4.x | Yes |
| < 0.4 | No |

## Reporting a Vulnerability

Please do not report security vulnerabilities through public GitHub issues.

**Report vulnerabilities using GitHub private vulnerability reporting.**

Include:

- a description of the vulnerability
- affected version or commit
- steps to reproduce
- expected impact
- any suggested fix, if known

## Vulnerability Handling Expectations

The maintainer will review private reports, ask follow-up questions when needed, and prioritize fixes according to severity and exploitability. Security fixes should stay narrowly scoped, include tests or verification where practical, and avoid changing deployment assumptions without explicit documentation.

Because this project is not yet public-production-ready, response timelines are best-effort.

## Security Scope

Reports are in scope when they affect the current backend, documentation, or deployment guidance, including:

- private `/v1` route exposure
- public incident viewer read-only access
- viewer/incident token leakage
- raw token logging
- request body logging
- uploaded file byte logging
- Authorization header logging
- upload size limits
- SHA-256 verification
- immutable chunk storage
- media stream completion validation
- ZIP bundle path traversal
- ZIP entry name safety
- filesystem path disclosure
- Docker bind exposure
- reverse proxy/TLS deployment
- evidence retention/deletion policy
- documentation that could mislead users about emergency-services contact, legal reporting, production readiness, or access-control guarantees

## Out-of-Scope Reports

The following are generally out of scope unless they demonstrate a concrete vulnerability in this repository:

- missing features already documented as absent, such as user accounts, OAuth, JWT, SMS, push notifications, trusted-contact accounts, Android/iOS clients, a web client, first-class incident types, escalation policies, or a public admin dashboard
- lack of production hardening already documented as a known limitation, without a new exploit path
- reports requiring public exposure of the private `/v1` API contrary to documented deployment guidance
- denial-of-service reports based only on unrealistic local access or unbounded physical access
- findings in future clients, recording implementations, account systems, notification systems, or key-sharing systems that are not in this repository
- legal admissibility, recording-law, or emergency-response claims that are not implemented behavior in this repository
- social engineering, phishing, or attacks against third-party hosting accounts

## Public Disclosure Guidance

Please allow time for private triage and remediation before public disclosure. Do not publish raw viewer tokens, incident tokens, request bodies, uploaded bytes, private deployment details, proof-of-concept material, or user safety data.