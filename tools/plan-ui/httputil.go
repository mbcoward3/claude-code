package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}

// writeSSE writes a single server-sent event frame.
func writeSSE(w http.ResponseWriter, ev sseEvent) {
	if ev.Event != "" {
		_, _ = w.Write([]byte("event: " + ev.Event + "\n"))
	}
	// Data may contain newlines; emit one data: line per line.
	for _, line := range strings.Split(ev.Data, "\n") {
		_, _ = w.Write([]byte("data: " + line + "\n"))
	}
	_, _ = w.Write([]byte("\n"))
}

// injectShellKey replaces the placeholder in shell.html with the live session key.
func injectShellKey(shell, key string) string {
	return strings.ReplaceAll(shell, "__PLAN_UI_KEY__", key)
}

// logRequests is a minimal pass-through; verbose logging can be added behind a flag.
func logRequests(next http.Handler) http.Handler {
	return next
}
