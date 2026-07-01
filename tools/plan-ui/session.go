package main

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// Session status values.
const (
	StatusOpen     = "open"     // live, no pending human feedback
	StatusFeedback = "feedback" // human has queued prompts waiting for the agent
	StatusEnded    = "ended"    // session closed
)

// Gate status values.
const (
	GatePending = "pending" // layout audit has not passed yet; artifact is masked
	GatePassed  = "passed"  // layout audit passed; artifact is visible
	GateFailed  = "failed"  // layout audit found blocking issues
)

// Agent presence values, surfaced in the browser UI.
const (
	PresenceWaiting   = "waiting"   // no agent attached
	PresenceListening = "listening" // agent is polling for feedback
	PresenceWorking   = "working"   // agent has feedback and is acting on it
)

// Target locates a human annotation within the artifact DOM.
type Target struct {
	Selector   string `json:"selector,omitempty"`
	QuotedText string `json:"quoted_text,omitempty"`
	Start      int    `json:"start,omitempty"`
	End        int    `json:"end,omitempty"`
}

// Prompt is a single piece of queued human feedback.
type Prompt struct {
	Text   string  `json:"text"`
	Action string  `json:"action,omitempty"` // "comment" | "approve" | "request-changes" | custom action id
	Target *Target `json:"target,omitempty"`
	TS     string  `json:"ts"`
}

// Message is one entry in the conversation history.
type Message struct {
	Role    string  `json:"role"` // "agent" | "human"
	Content string  `json:"content"`
	Action  string  `json:"action,omitempty"`
	Target  *Target `json:"target,omitempty"`
	TS      string  `json:"ts"`
}

// LayoutWarning is a single issue found by the browser-side layout audit.
type LayoutWarning struct {
	Type     string `json:"type"` // "overflow" | "clipped" | "overlap"
	Selector string `json:"selector"`
	Detail   string `json:"detail,omitempty"`
}

// Session is the persisted state for one plan artifact.
type Session struct {
	Key            string          `json:"key"`
	File           string          `json:"file"`
	URL            string          `json:"url"`
	Status         string          `json:"status"`
	Gate           string          `json:"gate"`
	Presence       string          `json:"presence"`
	PendingPrompts []Prompt        `json:"pending_prompts"`
	LayoutWarnings []LayoutWarning `json:"layout_warnings"`
	Chat           []Message       `json:"chat"`
	UpdatedAt      string          `json:"updated_at"`
}

// Store holds all sessions plus the live coordination primitives that are not
// persisted (SSE subscribers and long-poll waiters).
type Store struct {
	mu       sync.Mutex
	sessions map[string]*Session

	// live coordination, keyed by session key
	pollWaiters map[string][]chan struct{}
	sseClients  map[string][]chan sseEvent
}

// sseEvent is a server-sent event pushed to browser clients.
type sseEvent struct {
	Event string
	Data  string
}

func nowTS() string { return time.Now().UTC().Format(time.RFC3339) }

// NewStore loads persisted sessions from disk (if any) and returns a ready Store.
func NewStore() (*Store, error) {
	s := &Store{
		sessions:    map[string]*Session{},
		pollWaiters: map[string][]chan struct{}{},
		sseClients:  map[string][]chan sseEvent{},
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	path, err := statePath()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var persisted map[string]*Session
	if err := json.Unmarshal(data, &persisted); err != nil {
		// A corrupt state file should not wedge the tool; start fresh.
		return nil
	}
	if persisted != nil {
		s.sessions = persisted
	}
	return nil
}

// persist writes the current sessions to disk. Caller must hold s.mu.
func (s *Store) persist() error {
	path, err := statePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.sessions, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Get returns a copy-safe pointer to a session by key, or nil.
func (s *Store) Get(key string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessions[key]
}

// Upsert creates or resumes a session for a canonical file path and returns it.
func (s *Store) Upsert(canonical, url string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := keyForFile(canonical)
	sess := s.sessions[key]
	if sess == nil {
		sess = &Session{
			Key:            key,
			File:           canonical,
			PendingPrompts: []Prompt{},
			LayoutWarnings: []LayoutWarning{},
			Chat:           []Message{},
		}
		s.sessions[key] = sess
	}
	sess.URL = url
	sess.Status = StatusOpen
	sess.Gate = GatePending
	sess.Presence = PresenceWaiting
	sess.UpdatedAt = nowTS()
	_ = s.persist()
	return sess
}

// End marks a session ended and returns it (or nil if unknown).
func (s *Store) End(key string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.sessions[key]
	if sess == nil {
		return nil
	}
	sess.Status = StatusEnded
	sess.Presence = PresenceWaiting
	sess.UpdatedAt = nowTS()
	_ = s.persist()
	return sess
}

// QueuePrompts appends human feedback and wakes any waiting agent poll.
func (s *Store) QueuePrompts(key string, prompts []Prompt) {
	s.mu.Lock()
	sess := s.sessions[key]
	if sess == nil {
		s.mu.Unlock()
		return
	}
	for _, p := range prompts {
		if p.TS == "" {
			p.TS = nowTS()
		}
		sess.PendingPrompts = append(sess.PendingPrompts, p)
		sess.Chat = append(sess.Chat, Message{
			Role: "human", Content: p.Text, Action: p.Action, Target: p.Target, TS: p.TS,
		})
	}
	if sess.Status != StatusEnded {
		sess.Status = StatusFeedback
	}
	sess.UpdatedAt = nowTS()
	_ = s.persist()
	s.mu.Unlock()
	s.wakePollers(key)
}

// TakeFeedback drains pending prompts and layout warnings for the agent.
func (s *Store) TakeFeedback(key string) ([]Prompt, []LayoutWarning) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.sessions[key]
	if sess == nil {
		return nil, nil
	}
	prompts := sess.PendingPrompts
	warnings := sess.LayoutWarnings
	sess.PendingPrompts = []Prompt{}
	sess.LayoutWarnings = []LayoutWarning{}
	if sess.Status != StatusEnded {
		sess.Status = StatusOpen
	}
	sess.UpdatedAt = nowTS()
	_ = s.persist()
	return prompts, warnings
}

// HasFeedback reports whether an agent poll should return immediately.
func (s *Store) HasFeedback(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.sessions[key]
	if sess == nil {
		return false
	}
	return len(sess.PendingPrompts) > 0 || len(sess.LayoutWarnings) > 0 || sess.Status == StatusEnded
}

// AddAgentReply appends an agent message to the conversation and notifies browsers.
func (s *Store) AddAgentReply(key, text string) {
	s.mu.Lock()
	sess := s.sessions[key]
	if sess == nil {
		s.mu.Unlock()
		return
	}
	sess.Chat = append(sess.Chat, Message{Role: "agent", Content: text, TS: nowTS()})
	sess.UpdatedAt = nowTS()
	_ = s.persist()
	s.mu.Unlock()
	s.broadcast(key, sseEvent{Event: "agent-reply", Data: text})
}

// SetPresence updates the agent presence indicator and notifies browsers.
func (s *Store) SetPresence(key, presence string) {
	s.mu.Lock()
	sess := s.sessions[key]
	if sess == nil {
		s.mu.Unlock()
		return
	}
	sess.Presence = presence
	sess.UpdatedAt = nowTS()
	s.mu.Unlock()
	s.broadcast(key, sseEvent{Event: "presence", Data: presence})
}
