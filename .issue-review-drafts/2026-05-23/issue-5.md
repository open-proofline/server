# Issue #5: Update Emergency Viewer DOM During Polling

## Recommendation

keep-open

## Confidence

high

## Summary

The issue is still valid. The emergency viewer polls JSON, but the JavaScript still only stores the fetched data on `window.__lastEmergencyData`.

## Evidence reviewed

- Issue acceptance criteria:
  - Polling updates incident status and last-updated/check-in fields.
  - Polling updates chunk counts/latest chunk rows.
  - Polling updates completed recording download rows when streams complete.
  - Empty, failed, and completed states remain clear.
  - Tests cover rendered page/data behavior or JavaScript update helpers where practical.
  - DOM updates use safe assignment such as `textContent`.
- Relevant files:
  - `server/internal/httpapi/web/static/scripts.js:17` defines `poll`.
  - `server/internal/httpapi/web/static/scripts.js:24` stores data on `window.__lastEmergencyData`.
  - `server/internal/httpapi/web/static/scripts.js:30` initializes download links only once.
  - `server/internal/httpapi/web/templates/emergency.html:20` renders incident status statically.
  - `server/internal/httpapi/web/templates/emergency.html:31` renders latest checkin statically.
  - `server/internal/httpapi/web/templates/emergency.html:43` renders completed streams statically.
  - `server/internal/httpapi/web/templates/emergency.html:66` renders media chunk rows statically.
- Relevant commits or PRs:
  - No commit found that implements visible DOM polling updates.

## Analysis

The current JavaScript fetches `/data` every 10 seconds and retains the response for debugging/inspection, but it does not bind fetched values back into the visible page. Template-rendered status, checkins, completed streams, and chunk tables remain static after the initial page load.

## Suggested maintainer action

Keep the issue open. Add a small no-framework DOM update layer and focused tests or helper tests where practical.

## Draft comment

Reviewed against current `main`. This still appears valid: `scripts.js` polls `/data`, but the success path only assigns `window.__lastEmergencyData = data` and does not update the visible incident, checkin, completed recording, or chunk table DOM.

## Safe to close automatically?

no

## Notes

No sensitive details found in the issue body or review evidence.
