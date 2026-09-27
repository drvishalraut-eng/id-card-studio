package server

import (
	"net"
	"net/http"
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
