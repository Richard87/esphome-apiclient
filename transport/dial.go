package transport

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// dialTCP connects to address (host:port) with a timeout.
// For .local (mDNS) hostnames, it performs a multicast DNS query using a
// pure-Go implementation so no cgo or platform-specific resolver is needed.
func dialTCP(address string, timeout time.Duration) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		host = address
		port = "6053"
	}

	if strings.HasSuffix(strings.ToLower(host), ".local") {
		ip, err := resolveMDNS(host, timeout)
		if err != nil {
			return nil, fmt.Errorf("mDNS lookup %s: %w", host, err)
		}
		address = net.JoinHostPort(ip, port)
	}

	return net.DialTimeout("tcp", address, timeout)
}

// resolveMDNS resolves a .local hostname to an IP address using multicast DNS.
// It joins the mDNS multicast group (224.0.0.251:5353) to reliably receive
// responses, since many devices reply via multicast rather than unicast.
func resolveMDNS(host string, timeout time.Duration) (string, error) {
	fqdn := dns.Fqdn(host)

	msg := new(dns.Msg)
	msg.Id = 0 // mDNS uses ID 0
	msg.RecursionDesired = false
	msg.Question = []dns.Question{{
		Name:   fqdn,
		Qtype:  dns.TypeA,
		Qclass: dns.ClassINET,
	}}

	buf, err := msg.Pack()
	if err != nil {
		return "", fmt.Errorf("failed to pack mDNS query: %w", err)
	}

	// Join the mDNS multicast group to receive responses.
	// net.ListenMulticastUDP sets SO_REUSEADDR/SO_REUSEPORT so this works
	// even if another mDNS daemon (avahi, Bonjour) is already on port 5353.
	mcastAddr := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251), Port: 5353}
	conn, err := net.ListenMulticastUDP("udp4", nil, mcastAddr)
	if err != nil {
		return "", fmt.Errorf("failed to join mDNS multicast group: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(timeout))

	// Send query to the multicast group.
	if _, err := conn.WriteTo(buf, mcastAddr); err != nil {
		return "", fmt.Errorf("failed to send mDNS query: %w", err)
	}

	// Read responses until we find a matching A/AAAA record or timeout.
	respBuf := make([]byte, 65536)
	for {
		n, _, err := conn.ReadFrom(respBuf)
		if err != nil {
			return "", fmt.Errorf("no mDNS response (timed out): %w", err)
		}

		var resp dns.Msg
		if err := resp.Unpack(respBuf[:n]); err != nil {
			continue // skip malformed packets
		}

		for _, ans := range resp.Answer {
			switch rr := ans.(type) {
			case *dns.A:
				if strings.EqualFold(rr.Hdr.Name, fqdn) {
					return rr.A.String(), nil
				}
			case *dns.AAAA:
				if strings.EqualFold(rr.Hdr.Name, fqdn) {
					return rr.AAAA.String(), nil
				}
			}
		}
	}
}
