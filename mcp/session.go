package mcp

import (
	"sync"
	"time"
)

// Session holds MCP session state for Streamable HTTP.
type Session struct {
	ID              string
	ProtocolVersion string
	ClientInfo      *ClientInfo
	CreatedAt       time.Time
}

// SessionStore tracks active MCP sessions keyed by session ID.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewSessionStore creates an empty session store.
func NewSessionStore() *SessionStore {
	return &SessionStore{sessions: make(map[string]*Session)}
}

// Create registers a new session and returns it.
func (s *SessionStore) Create(protocolVersion string, clientInfo *ClientInfo) *Session {
	session := &Session{
		ID:              newSessionID(),
		ProtocolVersion: protocolVersion,
		ClientInfo:      clientInfo,
		CreatedAt:       time.Now(),
	}
	s.mu.Lock()
	s.sessions[session.ID] = session
	s.mu.Unlock()
	return session
}

// Get returns the session for the given ID, or nil if not found.
func (s *SessionStore) Get(id string) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[id]
}

// Delete removes a session.
func (s *SessionStore) Delete(id string) {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}

func newSessionID() string {
	return randomSessionID()
}
