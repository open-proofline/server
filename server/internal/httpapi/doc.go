// Package httpapi exposes separate v0.2.0 HTTP handlers for the private write
// API and the read-only emergency viewer.
//
// This package deliberately does not implement public authentication. The
// private handler is meant to run behind localhost, WireGuard, or firewall
// rules; adding user login, OAuth, JWT, or token sharing here would exceed that
// scope.
package httpapi
