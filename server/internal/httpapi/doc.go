// Package httpapi exposes the v0.1 private HTTP API for incidents, chunks,
// and checkins.
//
// This package deliberately does not implement public authentication. The
// v0.1 server is meant to run behind localhost, WireGuard, or firewall rules;
// adding user login, OAuth, JWT, or token sharing here would exceed that scope.
package httpapi
