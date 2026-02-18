package main

import (
	"slices"
	"sync"
	"time"
)

// store holds in-memory presence state: online users and typing status.
type store struct {
	mu          sync.RWMutex
	online      map[string]int       // username -> connection count
	typing      map[string]time.Time // username -> last typing timestamp
	cleanupDone chan struct{}
}

func newStore() *store {
	s := &store{
		online:      make(map[string]int),
		typing:      make(map[string]time.Time),
		cleanupDone: make(chan struct{}),
	}
	go s.cleanupTyping()
	return s
}

// connect increments the connection count and returns the current online users.
func (s *store) connect(username string) []string {
	s.mu.Lock()
	s.online[username]++
	users := s.onlineUsersLocked()
	s.mu.Unlock()
	return users
}

// disconnect decrements the connection count, removing the user if it reaches zero.
func (s *store) disconnect(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if count, exists := s.online[username]; exists && count <= 1 {
		delete(s.online, username)
		delete(s.typing, username)
	} else if exists {
		s.online[username]--
	}
}

func (s *store) setTyping(username string, isTyping bool) {
	s.mu.Lock()
	if isTyping {
		s.typing[username] = time.Now()
	} else {
		delete(s.typing, username)
	}
	s.mu.Unlock()
}

func (s *store) onlineUsers() []string {
	s.mu.RLock()
	users := s.onlineUsersLocked()
	s.mu.RUnlock()
	return users
}

// onlineUsersLocked returns sorted online usernames. Caller must hold s.mu.
func (s *store) onlineUsersLocked() []string {
	users := make([]string, 0, len(s.online))
	for u := range s.online {
		users = append(users, u)
	}
	slices.Sort(users)
	return users
}

func (s *store) typingUsers() []string {
	s.mu.RLock()
	users := make([]string, 0, len(s.typing))
	for u := range s.typing {
		users = append(users, u)
	}
	s.mu.RUnlock()
	slices.Sort(users)
	return users
}

// cleanupTyping periodically removes expired typing statuses.
func (s *store) cleanupTyping() {
	ticker := time.NewTicker(time.Second)
	defer func() {
		ticker.Stop()
		close(s.cleanupDone)
	}()

	for {
		select {
		case <-ticker.C:
			s.cleanupExpiredTyping()
		case <-s.cleanupDone:
			return
		}
	}
}

func (s *store) cleanupExpiredTyping() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for u, t := range s.typing {
		if now.Sub(t) > 5*time.Second {
			delete(s.typing, u)
		}
	}
}

func (s *store) stopCleanup() {
	close(s.cleanupDone)
}
