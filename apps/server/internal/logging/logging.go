// Package logging builds the structured logger used across the server.
//
// Output is JSON on stdout so a container log driver can parse it. Nothing in
// this server logs tokens, passwords or row payloads: the sync bodies are user
// calendar data and the credentials are secrets. Log identifiers and counts.
package logging

import (
	"io"
	"log/slog"
)

// New returns a JSON logger writing to w at the given minimum level.
func New(level slog.Level, w io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level}))
}
