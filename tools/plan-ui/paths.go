package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
)

// baseDir returns the plan-ui home directory (~/.plan-ui), creating it if needed.
func baseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".plan-ui")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// statePath is where the session store is persisted.
func statePath() (string, error) {
	dir, err := baseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "state.json"), nil
}

// serverInfoPath records the running server's pid and port.
func serverInfoPath() (string, error) {
	dir, err := baseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "server.json"), nil
}

// canonicalFile resolves a user-supplied path to an absolute, symlink-free path.
// This is the stable identity of a session.
func canonicalFile(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	// EvalSymlinks fails if the file does not exist yet; fall back to abs.
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved, nil
	}
	return abs, nil
}

// keyForFile derives a short, stable session key from a canonical file path.
func keyForFile(canonical string) string {
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])[:16]
}
