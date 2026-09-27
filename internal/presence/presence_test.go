package presence

import (
	"testing"
	"time"
)

func TestBeatThenListReturnsUser(t *testing.T) {
	m := New()
	m.Beat("ada", "192.0.2.10", "clients")

	users := m.List()
	if len(users) != 1 {
		t.Fatalf("got %d users, want 1", len(users))
	}
	u := users[0]
	if u.Username != "ada" || u.IP != "192.0.2.10" || u.Step != "clients" {
		t.Fatalf("got %+v", u)
	}
	if u.Host == "" {
		t.Fatal("expected a non-empty host, at least the IP fallback")
	}
}

func TestBeatOverwritesPreviousEntryForSameUser(t *testing.T) {
	m := New()
	m.Beat("ada", "192.0.2.10", "clients")
	m.Beat("ada", "192.0.2.10", "employees")

	users := m.List()
	if len(users) != 1 {
		t.Fatalf("got %d users, want 1 (same user, later heartbeat)", len(users))
	}
	if users[0].Step != "employees" {
		t.Fatalf("got step %q, want the latest heartbeat's step", users[0].Step)
	}
}

func TestListDropsUsersAfterSilenceTimeout(t *testing.T) {
	m := New()
	m.Beat("ada", "192.0.2.10", "clients")

	m.mu.Lock()
	e := m.users["ada"]
	e.LastSeen = time.Now().Add(-SilenceTimeout - time.Second)
	m.users["ada"] = e
	m.mu.Unlock()

	users := m.List()
	if len(users) != 0 {
		t.Fatalf("got %d users, want 0 after silence timeout", len(users))
	}
}

func TestListForgetsDroppedUsers(t *testing.T) {
	m := New()
	m.Beat("ada", "192.0.2.10", "clients")
	m.mu.Lock()
	e := m.users["ada"]
	e.LastSeen = time.Now().Add(-SilenceTimeout - time.Second)
	m.users["ada"] = e
	m.mu.Unlock()

	m.List() // triggers the drop

	m.mu.Lock()
	_, stillThere := m.users["ada"]
	m.mu.Unlock()
	if stillThere {
		t.Fatal("expected a stale user to be removed from the internal map, not just filtered")
	}
}

func TestResolveHostCachesResult(t *testing.T) {
	m := New()
	first := m.ResolveHost("192.0.2.55")

	m.mu.Lock()
	m.hosts["192.0.2.55"] = hostCacheEntry{host: "cached-value", expiresAt: time.Now().Add(hostCacheTTL)}
	m.mu.Unlock()

	second := m.ResolveHost("192.0.2.55")
	if second != "cached-value" {
		t.Fatalf("got %q, want the cached value (first lookup was %q)", second, first)
	}
}

func TestResolveHostFallsBackToIP(t *testing.T) {
	m := New()
	// 192.0.2.0/24 is TEST-NET-1 (RFC 5737): guaranteed not to reverse-resolve.
	host := m.ResolveHost("192.0.2.123")
	if host != "192.0.2.123" {
		t.Fatalf("got %q, want the IP itself as a fallback", host)
	}
}
