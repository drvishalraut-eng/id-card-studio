package server

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"
)

// clientIP extracts the caller's IP from r, stripping the port that
// http.Request.RemoteAddr always includes.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// resolveHost reverse-resolves ip to a hostname for the activity log,
// falling back to ip itself if the lookup fails or times out. It is not
// cached: activity-log writes are infrequent (sign-ins, admin actions),
// unlike the presence heartbeat, which caches lookups separately.
func resolveHost(ip string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	names, err := net.DefaultResolver.LookupAddr(ctx, ip)
	if err != nil || len(names) == 0 {
		return ip
	}
	return strings.TrimSuffix(names[0], ".")
}
