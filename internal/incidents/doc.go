// Package incidents owns the incident metadata model and SQLite repository.
//
// The repository records incidents, uploaded chunk metadata, and checkins while
// relying on SQLite constraints to prevent duplicate chunk indexes per incident
// and media type.
package incidents
