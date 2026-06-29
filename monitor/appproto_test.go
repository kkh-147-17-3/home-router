package monitor

import (
	"encoding/binary"
	"testing"
)

// ipv4TCP builds a minimal IPv4+TCP packet carrying the given payload,
// from src:sport to dst:dport, as NFLOG would deliver it (starting at L3).
func ipv4TCP(srcPort, dstPort int, payload []byte) []byte {
	ip := make([]byte, 20)
	ip[0] = 0x45 // version 4, IHL 5
	ip[9] = 6    // TCP
	copy(ip[12:16], []byte{203, 0, 113, 7}) // src 203.0.113.7
	copy(ip[16:20], []byte{192, 168, 1, 50}) // dst 192.168.1.50

	tcp := make([]byte, 20)
	binary.BigEndian.PutUint16(tcp[0:2], uint16(srcPort))
	binary.BigEndian.PutUint16(tcp[2:4], uint16(dstPort))
	tcp[12] = 5 << 4 // data offset 5 (20 bytes)

	return append(append(ip, tcp...), payload...)
}

// clientHello builds a minimal TLS ClientHello record with the given SNI.
func clientHello(sni string) []byte {
	name := []byte(sni)

	// server_name entry: type(0) + len(2) + name
	entry := append([]byte{0x00}, lenU16(len(name))...)
	entry = append(entry, name...)
	// server_name_list: list_len(2) + entry
	snList := append(lenU16(len(entry)), entry...)
	// extension: type(0x0000) + len(2) + snList
	ext := append([]byte{0x00, 0x00}, lenU16(len(snList))...)
	ext = append(ext, snList...)
	// extensions block: total_len(2) + ext
	exts := append(lenU16(len(ext)), ext...)

	body := []byte{0x03, 0x03}              // client_version
	body = append(body, make([]byte, 32)...) // random
	body = append(body, 0x00)                // session_id len 0
	body = append(body, 0x00, 0x02, 0x00, 0x2f) // cipher_suites len 2 + one suite
	body = append(body, 0x01, 0x00)          // compression len 1 + null
	body = append(body, exts...)

	// handshake: type(0x01) + len(3) + body
	hs := append([]byte{0x01}, lenU24(len(body))...)
	hs = append(hs, body...)

	// record: type(0x16) + version(2) + len(2) + hs
	rec := append([]byte{0x16, 0x03, 0x01}, lenU16(len(hs))...)
	rec = append(rec, hs...)
	return rec
}

func lenU16(n int) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(n))
	return b
}
func lenU24(n int) []byte {
	return []byte{byte(n >> 16), byte(n >> 8), byte(n)}
}

func TestParsePacketHTTP(t *testing.T) {
	payload := []byte("GET /admin/index.html?x=1 HTTP/1.1\r\nHost: nas.example.com\r\nUser-Agent: curl\r\n\r\n")
	pkt, ok := parsePacket(ipv4TCP(54321, 80, payload))
	if !ok {
		t.Fatal("parsePacket failed")
	}
	if pkt.SrcIP != "203.0.113.7" || pkt.DstIP != "192.168.1.50" {
		t.Fatalf("unexpected IPs: %s -> %s", pkt.SrcIP, pkt.DstIP)
	}
	if pkt.SrcPort != 54321 || pkt.DstPort != 80 {
		t.Fatalf("unexpected ports: %d -> %d", pkt.SrcPort, pkt.DstPort)
	}

	info, isHTTP := parseHTTPRequest(pkt.Payload)
	if !isHTTP {
		t.Fatal("expected HTTP request")
	}
	if info.Host != "nas.example.com" {
		t.Errorf("host = %q, want nas.example.com", info.Host)
	}
	if info.URL != "GET /admin/index.html?x=1" {
		t.Errorf("url = %q", info.URL)
	}
}

func TestParseHTTPClientIP(t *testing.T) {
	// Cloudflare (Flexible mode) forwards the real client via CF-Connecting-IP.
	cf := []byte("GET / HTTP/1.1\r\nHost: app.example.com\r\nCF-Connecting-IP: 198.51.100.23\r\nX-Forwarded-For: 198.51.100.23, 172.71.0.5\r\n\r\n")
	info, ok := parseHTTPRequest(cf)
	if !ok {
		t.Fatal("expected HTTP request")
	}
	if info.ClientIP != "198.51.100.23" {
		t.Errorf("clientIP = %q, want 198.51.100.23 (CF-Connecting-IP)", info.ClientIP)
	}

	// Without CF header, fall back to the first X-Forwarded-For hop.
	xff := []byte("GET / HTTP/1.1\r\nHost: app.example.com\r\nX-Forwarded-For: 203.0.113.99, 10.0.0.1\r\n\r\n")
	info, _ = parseHTTPRequest(xff)
	if info.ClientIP != "203.0.113.99" {
		t.Errorf("clientIP = %q, want 203.0.113.99 (first XFF hop)", info.ClientIP)
	}
}

func TestParsePacketTLSSNI(t *testing.T) {
	pkt, ok := parsePacket(ipv4TCP(40000, 443, clientHello("secure.example.com")))
	if !ok {
		t.Fatal("parsePacket failed")
	}
	if pkt.DstPort != 443 {
		t.Fatalf("dport = %d", pkt.DstPort)
	}
	if _, isHTTP := parseHTTPRequest(pkt.Payload); isHTTP {
		t.Fatal("TLS payload should not parse as HTTP")
	}
	sni, ok := parseTLSServerName(pkt.Payload)
	if !ok {
		t.Fatal("expected SNI")
	}
	if sni != "secure.example.com" {
		t.Errorf("sni = %q", sni)
	}
}

func TestParseNonApplicationPayload(t *testing.T) {
	// Bare ACK with no payload.
	if _, isHTTP := parseHTTPRequest(nil); isHTTP {
		t.Error("empty payload parsed as HTTP")
	}
	if _, ok := parseTLSServerName([]byte{0x17, 0x03, 0x03}); ok {
		t.Error("application-data record parsed as ClientHello")
	}
	// Truncated/garbage packet must not panic and must report not-ok.
	if _, ok := parsePacket([]byte{0x45, 0x00}); ok {
		t.Error("truncated packet parsed ok")
	}
}
