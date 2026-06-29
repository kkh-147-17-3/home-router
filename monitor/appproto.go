package monitor

import (
	"encoding/binary"
	"net"
	"strings"
)

// l4Packet is the L3/L4 metadata extracted from a raw NFLOG payload.
type l4Packet struct {
	SrcIP   string
	DstIP   string
	SrcPort int
	DstPort int
	Proto   string // "tcp" / "udp"
	Payload []byte // L4 payload (TCP/UDP data), may be empty
}

// parsePacket decodes an IPv4/IPv6 packet (starting at the network header, as
// delivered by NFLOG) far enough to extract the 5-tuple and L4 payload.
// Only TCP/UDP are handled; everything else returns ok=false.
func parsePacket(data []byte) (l4Packet, bool) {
	if len(data) < 20 {
		return l4Packet{}, false
	}

	version := data[0] >> 4
	var (
		l4         []byte
		protoNum   byte
		src, dst   net.IP
	)

	switch version {
	case 4:
		ihl := int(data[0]&0x0f) * 4
		if ihl < 20 || len(data) < ihl {
			return l4Packet{}, false
		}
		protoNum = data[9]
		src = net.IP(data[12:16])
		dst = net.IP(data[16:20])
		l4 = data[ihl:]
	case 6:
		if len(data) < 40 {
			return l4Packet{}, false
		}
		protoNum = data[6] // next header; extension headers not handled
		src = net.IP(data[8:24])
		dst = net.IP(data[24:40])
		l4 = data[40:]
	default:
		return l4Packet{}, false
	}

	pkt := l4Packet{SrcIP: src.String(), DstIP: dst.String()}

	switch protoNum {
	case 6: // TCP
		if len(l4) < 20 {
			return l4Packet{}, false
		}
		dataOff := int(l4[12]>>4) * 4
		if dataOff < 20 || len(l4) < dataOff {
			return l4Packet{}, false
		}
		pkt.Proto = "tcp"
		pkt.SrcPort = int(binary.BigEndian.Uint16(l4[0:2]))
		pkt.DstPort = int(binary.BigEndian.Uint16(l4[2:4]))
		pkt.Payload = l4[dataOff:]
	case 17: // UDP
		if len(l4) < 8 {
			return l4Packet{}, false
		}
		pkt.Proto = "udp"
		pkt.SrcPort = int(binary.BigEndian.Uint16(l4[0:2]))
		pkt.DstPort = int(binary.BigEndian.Uint16(l4[2:4]))
		pkt.Payload = l4[8:]
	default:
		return l4Packet{}, false
	}

	return pkt, true
}

var httpMethods = []string{"GET ", "POST ", "PUT ", "DELETE ", "HEAD ", "OPTIONS ", "PATCH ", "CONNECT ", "TRACE "}

// httpRequestInfo is the application-layer data extracted from a plaintext
// HTTP request.
type httpRequestInfo struct {
	Host     string // Host header
	URL      string // "METHOD path"
	ClientIP string // real client IP from a proxy header (CF-Connecting-IP / X-Forwarded-For), if any
}

// parseHTTPRequest extracts request info from a plaintext HTTP request payload.
// Returns ok=false if the payload is not an HTTP request.
func parseHTTPRequest(payload []byte) (httpRequestInfo, bool) {
	if len(payload) < 16 {
		return httpRequestInfo{}, false
	}

	isHTTP := false
	for _, m := range httpMethods {
		if len(payload) >= len(m) && string(payload[:len(m)]) == m {
			isHTTP = true
			break
		}
	}
	if !isHTTP {
		return httpRequestInfo{}, false
	}

	// Limit scan to the header block.
	head := payload
	if len(head) > 8192 {
		head = head[:8192]
	}
	text := string(head)

	lineEnd := strings.IndexByte(text, '\n')
	if lineEnd < 0 {
		return httpRequestInfo{}, false
	}
	requestLine := strings.TrimRight(text[:lineEnd], "\r")
	parts := strings.Fields(requestLine)
	if len(parts) < 2 {
		return httpRequestInfo{}, false
	}

	info := httpRequestInfo{URL: parts[0] + " " + parts[1]}

	// Scan headers (case-insensitive). CF-Connecting-IP wins over X-Forwarded-For.
	var xff string
	for _, line := range strings.Split(text[lineEnd+1:], "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			break // end of headers
		}
		colon := strings.IndexByte(line, ':')
		if colon <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:colon])
		val := strings.TrimSpace(line[colon+1:])
		switch {
		case strings.EqualFold(key, "Host"):
			info.Host = val
		case strings.EqualFold(key, "CF-Connecting-IP"):
			info.ClientIP = val
		case strings.EqualFold(key, "X-Forwarded-For"):
			// First hop is the original client.
			if i := strings.IndexByte(val, ','); i >= 0 {
				xff = strings.TrimSpace(val[:i])
			} else {
				xff = val
			}
		}
	}
	if info.ClientIP == "" {
		info.ClientIP = xff
	}

	return info, true
}

// parseTLSServerName extracts the SNI (server_name) from a TLS ClientHello
// payload. Returns ok=false if the payload is not a ClientHello or has no SNI.
func parseTLSServerName(payload []byte) (string, bool) {
	// TLS record header: type(1) version(2) length(2)
	if len(payload) < 5 || payload[0] != 0x16 { // 0x16 = handshake
		return "", false
	}
	rec := payload[5:]

	// Handshake header: type(1) length(3)
	if len(rec) < 4 || rec[0] != 0x01 { // 0x01 = ClientHello
		return "", false
	}
	hs := rec[4:]

	// client_version(2) + random(32)
	pos := 2 + 32
	if len(hs) < pos+1 {
		return "", false
	}
	// session_id
	sidLen := int(hs[pos])
	pos += 1 + sidLen
	if len(hs) < pos+2 {
		return "", false
	}
	// cipher_suites
	csLen := int(binary.BigEndian.Uint16(hs[pos : pos+2]))
	pos += 2 + csLen
	if len(hs) < pos+1 {
		return "", false
	}
	// compression_methods
	compLen := int(hs[pos])
	pos += 1 + compLen
	if len(hs) < pos+2 {
		return "", false
	}
	// extensions
	extLen := int(binary.BigEndian.Uint16(hs[pos : pos+2]))
	pos += 2
	if len(hs) < pos+extLen {
		extLen = len(hs) - pos // tolerate truncation
	}
	ext := hs[pos : pos+extLen]

	for len(ext) >= 4 {
		extType := binary.BigEndian.Uint16(ext[0:2])
		l := int(binary.BigEndian.Uint16(ext[2:4]))
		if len(ext) < 4+l {
			break
		}
		body := ext[4 : 4+l]
		ext = ext[4+l:]

		if extType != 0x0000 { // server_name
			continue
		}
		// server_name_list: list_len(2), then entries: type(1) len(2) name
		if len(body) < 2 {
			return "", false
		}
		list := body[2:]
		for len(list) >= 3 {
			nameType := list[0]
			nameLen := int(binary.BigEndian.Uint16(list[1:3]))
			if len(list) < 3+nameLen {
				break
			}
			if nameType == 0x00 { // host_name
				return string(list[3 : 3+nameLen]), true
			}
			list = list[3+nameLen:]
		}
		return "", false
	}

	return "", false
}
