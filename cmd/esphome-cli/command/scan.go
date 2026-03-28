package command

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

func RunScan(ctx context.Context, timeout time.Duration) error {

	// mDNS multicast address
	maddr, err := net.ResolveUDPAddr("udp4", "224.0.0.251:5353")
	if err != nil {
		return fmt.Errorf("failed to resolve mDNS address: %w", err)
	}

	// Use a random port for sending
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return fmt.Errorf("failed to listen on UDP: %w", err)
	}
	defer conn.Close()

	// Prepare the query
	msg := new(dns.Msg)
	msg.SetQuestion("_esphomelib._tcp.local.", dns.TypePTR)
	// Some mDNS implementations want the unicast-response bit set if we're not on port 5353
	msg.Question[0].Qclass |= 0x8000

	buf, err := msg.Pack()
	if err != nil {
		return fmt.Errorf("failed to pack DNS message: %w", err)
	}

	// Send the query
	if _, err := conn.WriteTo(buf, maddr); err != nil {
		return fmt.Errorf("failed to send mDNS query: %w", err)
	}

	fmt.Printf("Scanning for ESPHome devices for %v...\n", timeout)
	fmt.Printf("%-25s %-16s %-6s %s\n", "NAME", "ADDRESS", "PORT", "TXT")

	seen := make(map[string]bool)
	deadline := time.Now().Add(timeout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		conn.SetReadDeadline(deadline)
		readBuf := make([]byte, 2048)
		n, addr, err := conn.ReadFrom(readBuf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break
			}
			return fmt.Errorf("error reading from UDP: %w", err)
		}

		resp := new(dns.Msg)
		if err := resp.Unpack(readBuf[:n]); err != nil {
			continue
		}

		for _, rr := range resp.Answer {
			if ptr, ok := rr.(*dns.PTR); ok && strings.HasSuffix(ptr.Hdr.Name, "_esphomelib._tcp.local.") {
				instance := ptr.Ptr
				if seen[instance] {
					continue
				}

				// Find related records
				var ip net.IP
				if udpAddr, ok := addr.(*net.UDPAddr); ok {
					ip = udpAddr.IP
				}

				var port uint16 = 6053 // Default ESPHome port
				var txt []string

				// Check Answer and Extra for SRV, A, and TXT
				allRecords := append(resp.Answer, resp.Extra...)

				// First pass: find SRV and TXT
				var srvTarget string
				for _, r := range allRecords {
					switch record := r.(type) {
					case *dns.SRV:
						if record.Hdr.Name == instance {
							port = record.Port
							srvTarget = record.Target
						}
					case *dns.TXT:
						if record.Hdr.Name == instance {
							txt = record.Txt
						}
					}
				}

				// Second pass: find A record (either matching instance or srvTarget)
				for _, r := range allRecords {
					if a, ok := r.(*dns.A); ok {
						if a.Hdr.Name == instance || (srvTarget != "" && a.Hdr.Name == srvTarget) {
							ip = a.A
							break
						}
					}
				}

				name := strings.TrimSuffix(instance, "._esphomelib._tcp.local.")
				fmt.Printf("%-25s %-16v %-6d %v\n", name, ip, port, txt)
				seen[instance] = true
			}
		}
	}

	return nil
}
