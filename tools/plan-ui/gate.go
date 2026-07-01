package main

// The layout gate keeps the artifact masked in the browser until an automated
// audit (overflow / clipped text / overlap) passes. When the audit fails, the
// warnings are surfaced to the agent on its next poll so it can fix the layout.

// ReportGate records the browser-side layout audit result for a session.
//   - passed: no blocking issues; the browser unmasks the artifact.
//   - failed: warnings are queued as feedback for the agent and the artifact
//     stays masked.
func (s *Store) ReportGate(key string, warnings []LayoutWarning) {
	s.mu.Lock()
	sess := s.sessions[key]
	if sess == nil {
		s.mu.Unlock()
		return
	}

	if len(warnings) == 0 {
		sess.Gate = GatePassed
		sess.UpdatedAt = nowTS()
		_ = s.persist()
		s.mu.Unlock()
		s.broadcast(key, sseEvent{Event: "gate", Data: GatePassed})
		return
	}

	sess.Gate = GateFailed
	sess.LayoutWarnings = warnings
	if sess.Status != StatusEnded {
		sess.Status = StatusFeedback
	}
	sess.UpdatedAt = nowTS()
	_ = s.persist()
	s.mu.Unlock()

	s.broadcast(key, sseEvent{Event: "gate", Data: GateFailed})
	// Wake the agent so it learns about the layout failure promptly.
	s.wakePollers(key)
}

// GateStatus returns the current gate state for a session (empty if unknown).
func (s *Store) GateStatus(key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.sessions[key]
	if sess == nil {
		return ""
	}
	return sess.Gate
}
