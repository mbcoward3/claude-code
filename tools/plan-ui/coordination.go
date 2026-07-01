package main

// This file holds the in-memory coordination between the agent long-poll and the
// browser SSE stream. None of it is persisted.

// subscribePoll registers a waiter that is closed when new feedback arrives.
func (s *Store) subscribePoll(key string) chan struct{} {
	ch := make(chan struct{}, 1)
	s.mu.Lock()
	s.pollWaiters[key] = append(s.pollWaiters[key], ch)
	s.mu.Unlock()
	return ch
}

// unsubscribePoll removes a waiter (e.g. when the poll's context is cancelled).
func (s *Store) unsubscribePoll(key string, ch chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	waiters := s.pollWaiters[key]
	for i, w := range waiters {
		if w == ch {
			s.pollWaiters[key] = append(waiters[:i], waiters[i+1:]...)
			break
		}
	}
}

// wakePollers signals every waiter for a session that feedback is available.
func (s *Store) wakePollers(key string) {
	s.mu.Lock()
	waiters := s.pollWaiters[key]
	s.pollWaiters[key] = nil
	s.mu.Unlock()
	for _, ch := range waiters {
		select {
		case ch <- struct{}{}:
		default:
		}
		close(ch)
	}
}

// subscribeSSE registers a browser client stream for a session.
func (s *Store) subscribeSSE(key string) chan sseEvent {
	ch := make(chan sseEvent, 16)
	s.mu.Lock()
	s.sseClients[key] = append(s.sseClients[key], ch)
	s.mu.Unlock()
	return ch
}

// unsubscribeSSE removes a browser client stream.
func (s *Store) unsubscribeSSE(key string, ch chan sseEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	clients := s.sseClients[key]
	for i, c := range clients {
		if c == ch {
			s.sseClients[key] = append(clients[:i], clients[i+1:]...)
			break
		}
	}
}

// broadcast pushes an event to every browser client for a session.
func (s *Store) broadcast(key string, ev sseEvent) {
	s.mu.Lock()
	clients := append([]chan sseEvent(nil), s.sseClients[key]...)
	s.mu.Unlock()
	for _, ch := range clients {
		select {
		case ch <- ev:
		default:
			// Drop events for a slow client rather than blocking the server.
		}
	}
}

// hasSSEClients reports whether any browser is currently connected to a session.
func (s *Store) hasSSEClients(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.sseClients[key]) > 0
}
