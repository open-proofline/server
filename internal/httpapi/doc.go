// Package httpapi exposes separate HTTP handlers for the private write API and
// the read-only incident viewer.
//
// This package implements local account/session authentication for the private
// handler. The private handler is still meant to run behind localhost,
// WireGuard, or firewall rules; adding public authentication, OAuth, JWT, or a
// public account portal here would exceed that scope.
package httpapi
