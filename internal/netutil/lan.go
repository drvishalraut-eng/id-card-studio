// Package netutil provides small network helpers shared across the server:
// discovering this host's LAN-reachable addresses.
package netutil

import (
	"fmt"
	"net"
)

// LANURLs returns "http://<ip>:<port>" for every non-loopback IPv4 address
// bound to this host, so operators can see every LAN URL the server is
// reachable at.
func LANURLs(port int) []string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}
	var urls []string
	for _, a := range addrs {
		ipNet, ok := a.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() {
			continue
		}
		ip4 := ipNet.IP.To4()
		if ip4 == nil {
			continue
		}
		urls = append(urls, fmt.Sprintf("http://%s:%d", ip4.String(), port))
	}
	return urls
}
