# Update Emergency Viewer DOM During Polling

## Priority

P2

## Type

bug

## Labels

- `backlog`
- `bug`
- `testing`

## Summary

The emergency viewer polls `/e/{token}/data`, but the fetched data is only stored on `window.__lastEmergencyData` and does not update visible page content. Make polling update the read-only viewer UI.

## Context

`docs/api.md` says `GET /e/{token}/data` returns the same read-only summary as JSON for polling. `server/internal/httpapi/web/static/scripts.js` polls the endpoint every 10 seconds but does not update the incident status, latest check-in, chunk table, completed-stream list, or download buttons.

## Proposed change

Add a small no-framework DOM update path for emergency viewer data. Keep routes read-only, keep the CSP compatible with external static JavaScript, and avoid exposing raw tokens outside the current URL-based flow.

## Acceptance criteria

- [ ] Polling updates incident status and last-updated/check-in fields.
- [ ] Polling updates chunk counts/latest chunk rows.
- [ ] Polling updates completed recording download rows when streams complete.
- [ ] Empty, failed, and completed states remain clear.
- [ ] Tests cover representative rendered page/data behavior or JavaScript update helpers where practical.

## Tests / validation

- [ ] `cd server && go test ./...`
- [ ] simulator smoke test, if relevant
- [ ] manual emergency viewer check in a browser, if JavaScript changes
- [ ] docs updated, if relevant

## Out of scope

Do not add React, Node, npm, a frontend build step, backend/browser decryption, playable exports, or write/mutation routes to the public emergency viewer.

## Notes

Related files: `server/internal/httpapi/web/static/scripts.js`, `server/internal/httpapi/web/templates/emergency.html`, `server/internal/httpapi/emergency.go`, `docs/api.md`.
