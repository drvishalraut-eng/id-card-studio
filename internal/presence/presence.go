// Package presence tracks which users are currently connected, from the
// periodic heartbeat each browser tab sends, for the "Under the hood" panel.
package presence

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"
)

// SilenceTimeout is how long without a heartbeat before a user is dropped.
const SilenceTimeout = 60 * time.Second

// hostCacheTTL is how long a reverse-DNS lookup result is reused before
// being looked up again.
const hostCacheTTL = 10 * time.Minute

// Entry is one connected user, as shown in the "Under the hood" panel.
type Entry struct {
	Username string    `json:"username"`
	IP       string    `json:"ip"`
	Host     string    `json:"host"`
	Step     string    `json:"step"`
	LastSeen time.Time `json:"last_seen"`
}

type hostCacheEntry struct {
	host      string
	expiresAt time.Time
}

// Manager tracks connected users in memory and reverse-resolves their IPs
// to hostnames, caching each result for 10 minutes and falling back to the
// IP itself when the lookup fails.
type Manager struct {
	mu    sync.Mutex
	users map[string]Entry
	hosts map[string]hostCacheEntry
}

// New returns an empty Manager.
func New() *Manager {
	return &Manager{
		users: make(map[string]Entry),
		hosts: make(map[string]hostCacheEntry),
	}
}

// Beat records a heartbeat for username at ip, currently on wizard step
// step.
func (m *Manager) Beat(username, ip, step string) {
	host := m.ResolveHost(ip)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[username] = Entry{Username: username, IP: ip, Host: host, Step: step, LastSeen: time.Now()}
}

// List returns every user seen within the last SilenceTimeout, dropping
// (and forgetting) anyone silent longer than that.
func (m *Manager) List() []Entry {
	m.mu.Lock()
	defer m.mu.Unlock()
	cutoff := time.Now().Add(-SilenceTimeout)
	out := make([]Entry, 0, len(m.users))
	for username, e := range m.users {
		if e.LastSeen.Before(cutoff) {
			delete(m.users, username)
			continue
		}
		out = append(out, e)
	}
	return out
}

// ResolveHost reverse-resolves ip to a hostname, caching the result (or the
// ip fallback, if the lookup fails or times out) for 10 minutes.
func (m *Manager) ResolveHost(ip string) string {
	m.mu.Lock()
	if c, ok := m.hosts[ip]; ok && time.Now().Before(c.expiresAt) {
		m.mu.Unlock()
		return c.host
	}
	m.mu.Unlock()

	host := ip
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if names, err := net.DefaultResolver.LookupAddr(ctx, ip); err == nil && len(names) > 0 {
		host = strings.TrimSuffix(names[0], ".")
	}

	m.mu.Lock()
	m.hosts[ip] = hostCacheEntry{host: host, expiresAt: time.Now().Add(hostCacheTTL)}
	m.mu.Unlock()
	return host
}
