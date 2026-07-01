package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// serverInfo is persisted so CLI client commands can find the running server.
type serverInfo struct {
	PID  int    `json:"pid"`
	Port int    `json:"port"`
	URL  string `json:"url"`
}

// Server hosts the local HTTP API, SSE streams, and static assets.
type Server struct {
	store   *Store
	http    *http.Server
	port    int
	baseURL string

	watchMu  sync.Mutex
	watching map[string]context.CancelFunc // key -> stop file watcher

	idleAfter time.Duration
	lastActive time.Time
	activeMu   sync.Mutex
}

// newServer binds a listener (port 0 = OS-assigned) and prepares routes.
func newServer(port int) (*Server, net.Listener, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		return nil, nil, err
	}
	actual := ln.Addr().(*net.TCPAddr).Port
	store, err := NewStore()
	if err != nil {
		return nil, nil, err
	}
	s := &Server{
		store:      store,
		port:       actual,
		baseURL:    fmt.Sprintf("http://127.0.0.1:%d", actual),
		watching:   map[string]context.CancelFunc{},
		idleAfter:  30 * time.Minute,
		lastActive: time.Now(),
	}
	s.http = &http.Server{Handler: s.routes()}
	return s, ln, nil
}

func (s *Server) touch() {
	s.activeMu.Lock()
	s.lastActive = time.Now()
	s.activeMu.Unlock()
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "port": s.port})
	})

	// Agent-facing API.
	mux.HandleFunc("POST /api/session", s.handleSession)
	mux.HandleFunc("POST /api/end", s.handleEnd)
	mux.HandleFunc("GET /api/poll", s.handlePoll)
	mux.HandleFunc("POST /api/stop", s.handleStop)

	// Session-scoped API (browser + agent).
	mux.HandleFunc("POST /api/{key}/agent-reply", s.handleAgentReply)
	mux.HandleFunc("POST /api/{key}/feedback", s.handleFeedback)
	mux.HandleFunc("POST /api/{key}/gate", s.handleGate)
	mux.HandleFunc("GET /api/{key}/state", s.handleState)

	// Browser surfaces.
	mux.HandleFunc("GET /s/{key}", s.handleShell)
	mux.HandleFunc("GET /artifact/{key}", s.handleArtifact)
	mux.HandleFunc("GET /events/{key}", s.handleEvents)
	mux.HandleFunc("GET /assets/{name}", s.handleAsset)

	return logRequests(mux)
}

// --- Agent API ---------------------------------------------------------------

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	s.touch()
	var req struct {
		File string `json:"file"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.File == "" {
		writeErr(w, http.StatusBadRequest, "missing file")
		return
	}
	canonical, err := canonicalFile(req.File)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := os.Stat(canonical); err != nil {
		writeErr(w, http.StatusBadRequest, "file does not exist: "+canonical)
		return
	}
	key := keyForFile(canonical)
	url := s.baseURL + "/s/" + key
	sess := s.store.Upsert(canonical, url)
	s.startWatch(sess.Key, canonical)

	writeJSON(w, http.StatusOK, map[string]any{
		"session":   sessionView(sess),
		"next_step": "Open " + url + " in a browser, then run `plan-ui poll " + req.File + "` to wait for feedback. The poll blocks silently until the human responds — never kill it.",
	})
}

func (s *Server) handleEnd(w http.ResponseWriter, r *http.Request) {
	s.touch()
	var req struct {
		File string `json:"file"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.File == "" {
		writeErr(w, http.StatusBadRequest, "missing file")
		return
	}
	canonical, err := canonicalFile(req.File)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	key := keyForFile(canonical)
	s.stopWatch(key)
	sess := s.store.End(key)
	if sess == nil {
		writeErr(w, http.StatusNotFound, "no session for file")
		return
	}
	s.store.broadcast(key, sseEvent{Event: "ended", Data: ""})
	writeJSON(w, http.StatusOK, map[string]any{
		"session":   sessionView(sess),
		"next_step": "Session ended.",
	})
}

// handlePoll long-polls for human feedback or layout warnings.
func (s *Server) handlePoll(w http.ResponseWriter, r *http.Request) {
	s.touch()
	file := r.URL.Query().Get("file")
	if file == "" {
		writeErr(w, http.StatusBadRequest, "missing file")
		return
	}
	canonical, err := canonicalFile(file)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	key := keyForFile(canonical)
	if s.store.Get(key) == nil {
		writeErr(w, http.StatusNotFound, "no session for file; run `plan-ui open` first")
		return
	}

	// Optional agent reply to display before waiting again.
	if reply := r.URL.Query().Get("agent_reply"); reply != "" {
		s.store.AddAgentReply(key, reply)
	}

	timeout := time.Duration(0)
	if ms := r.URL.Query().Get("timeout_ms"); ms != "" {
		if n, err := strconv.Atoi(ms); err == nil && n > 0 {
			timeout = time.Duration(n) * time.Millisecond
		}
	}

	s.store.SetPresence(key, PresenceListening)

	for {
		if s.store.HasFeedback(key) {
			s.store.SetPresence(key, PresenceWorking)
			prompts, warnings := s.store.TakeFeedback(key)
			sess := s.store.Get(key)
			writeJSON(w, http.StatusOK, map[string]any{
				"prompts":         prompts,
				"layout_warnings": warnings,
				"session":         sessionView(sess),
				"next_step":       pollNextStep(sess, file),
			})
			return
		}

		waiter := s.store.subscribePoll(key)
		var timeoutCh <-chan time.Time
		if timeout > 0 {
			t := time.NewTimer(timeout)
			defer t.Stop()
			timeoutCh = t.C
		}
		select {
		case <-waiter:
			// Loop: re-check for feedback.
		case <-timeoutCh:
			s.store.unsubscribePoll(key, waiter)
			s.store.SetPresence(key, PresenceWaiting)
			writeJSON(w, http.StatusOK, map[string]any{
				"prompts":         []Prompt{},
				"layout_warnings": []LayoutWarning{},
				"timed_out":       true,
				"next_step":       "No feedback yet. Poll again to keep waiting.",
			})
			return
		case <-r.Context().Done():
			s.store.unsubscribePoll(key, waiter)
			s.store.SetPresence(key, PresenceWaiting)
			return
		}
	}
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"server": map[string]any{"status": "stopping"}})
	go func() {
		time.Sleep(100 * time.Millisecond)
		s.shutdown()
	}()
}

// --- Session-scoped API ------------------------------------------------------

func (s *Server) handleAgentReply(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad body")
		return
	}
	s.store.AddAgentReply(key, req.Text)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleFeedback(w http.ResponseWriter, r *http.Request) {
	s.touch()
	key := r.PathValue("key")
	var req struct {
		Prompts []Prompt `json:"prompts"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad body")
		return
	}
	s.store.QueuePrompts(key, req.Prompts)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "queued": len(req.Prompts)})
}

func (s *Server) handleGate(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	var req struct {
		Warnings []LayoutWarning `json:"warnings"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad body")
		return
	}
	s.store.ReportGate(key, req.Warnings)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "gate": s.store.GateStatus(key)})
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	sess := s.store.Get(key)
	if sess == nil {
		writeErr(w, http.StatusNotFound, "unknown session")
		return
	}
	writeJSON(w, http.StatusOK, sessionView(sess))
}

// --- Browser surfaces --------------------------------------------------------

func (s *Server) handleShell(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	sess := s.store.Get(key)
	if sess == nil {
		http.NotFound(w, r)
		return
	}
	shell := mustAsset("shell.html")
	shell = injectShellKey(shell, key)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(shell))
}

func (s *Server) handleArtifact(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	sess := s.store.Get(key)
	if sess == nil {
		http.NotFound(w, r)
		return
	}
	html, err := transformArtifact(sess.File, key)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "cannot read artifact: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(html)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if s.store.Get(key) == nil {
		http.NotFound(w, r)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := s.store.subscribeSSE(key)
	defer s.store.unsubscribeSSE(key, ch)

	// Prime the client with current presence + gate so it renders immediately.
	if sess := s.store.Get(key); sess != nil {
		writeSSE(w, sseEvent{Event: "presence", Data: sess.Presence})
		writeSSE(w, sseEvent{Event: "gate", Data: sess.Gate})
		flusher.Flush()
	}

	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case ev := <-ch:
			writeSSE(w, ev)
			flusher.Flush()
		case <-heartbeat.C:
			_, _ = fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) handleAsset(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	data, err := asset(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", contentType(name))
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(data)
}

// --- File watching (live reload) ---------------------------------------------

// startWatch launches an mtime poller that broadcasts a reload event when the
// artifact file changes. Uses stdlib only (no fsnotify dependency).
func (s *Server) startWatch(key, canonical string) {
	s.watchMu.Lock()
	defer s.watchMu.Unlock()
	if _, ok := s.watching[key]; ok {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.watching[key] = cancel
	go func() {
		var last time.Time
		if fi, err := os.Stat(canonical); err == nil {
			last = fi.ModTime()
		}
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fi, err := os.Stat(canonical)
				if err != nil {
					continue
				}
				if fi.ModTime().After(last) {
					last = fi.ModTime()
					// A fresh edit invalidates the previous layout audit.
					s.store.broadcast(key, sseEvent{Event: "reload", Data: ""})
				}
			}
		}
	}()
}

func (s *Server) stopWatch(key string) {
	s.watchMu.Lock()
	defer s.watchMu.Unlock()
	if cancel, ok := s.watching[key]; ok {
		cancel()
		delete(s.watching, key)
	}
}

// --- Lifecycle ---------------------------------------------------------------

func (s *Server) serve(ln net.Listener) error {
	if err := writeServerInfo(serverInfo{PID: os.Getpid(), Port: s.port, URL: s.baseURL}); err != nil {
		return err
	}
	defer removeServerInfo()
	go s.idleLoop()
	err := s.http.Serve(ln)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = s.http.Shutdown(ctx)
}

// idleLoop shuts the server down after a period with no sessions and no activity.
func (s *Server) idleLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.activeMu.Lock()
		idle := time.Since(s.lastActive)
		s.activeMu.Unlock()
		if idle < s.idleAfter {
			continue
		}
		if s.store.anyOpenSession() {
			continue
		}
		s.shutdown()
		return
	}
}

// anyOpenSession reports whether any session is still open (not ended).
func (s *Store) anyOpenSession() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sess := range s.sessions {
		if sess.Status != StatusEnded {
			return true
		}
	}
	return false
}

// --- helpers -----------------------------------------------------------------

func writeServerInfo(info serverInfo) error {
	path, err := serverInfoPath()
	if err != nil {
		return err
	}
	data, _ := json.MarshalIndent(info, "", "  ")
	return os.WriteFile(path, data, 0o644)
}

func readServerInfo() (serverInfo, error) {
	var info serverInfo
	path, err := serverInfoPath()
	if err != nil {
		return info, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return info, err
	}
	err = json.Unmarshal(data, &info)
	return info, err
}

func removeServerInfo() {
	if path, err := serverInfoPath(); err == nil {
		_ = os.Remove(path)
	}
}

// sessionView is the client-facing projection of a session.
func sessionView(sess *Session) map[string]any {
	return map[string]any{
		"key":      sess.Key,
		"file":     sess.File,
		"url":      sess.URL,
		"status":   sess.Status,
		"gate":     sess.Gate,
		"presence": sess.Presence,
	}
}

func pollNextStep(sess *Session, file string) string {
	if sess != nil && sess.Status == StatusEnded {
		return "The session was ended by the human. Stop polling."
	}
	return "Apply the feedback to the plan file, then run `plan-ui poll " + file +
		" --agent-reply \"<summary of what you changed>\"` to show your response and wait again."
}
