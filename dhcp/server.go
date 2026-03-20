package dhcp

import (
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/server4"
	"golang.org/x/net/context"
	"home-router/internal/config"
	"log"
	"net"
)

func RunServer(ctx context.Context, iface string, pool *Pool, cfg *config.Config) error {
	handler := func(conn net.PacketConn, peer net.Addr, m *dhcpv4.DHCPv4) {
		msgType := m.MessageType()
		macAddress := m.ClientHWAddr
		hostname := m.HostName()
		logHostname := hostname
		if logHostname == "" {
			logHostname = "(알 수 없음)"
		}
		log.Printf("[DHCP Server] 수신: MAC=%s, 호스트=%s, 타입=%s", macAddress, logHostname, msgType)

		var replyType dhcpv4.MessageType
		switch msgType {
		case dhcpv4.MessageTypeDiscover:
			replyType = dhcpv4.MessageTypeOffer
		case dhcpv4.MessageTypeRequest:
			replyType = dhcpv4.MessageTypeAck
		case dhcpv4.MessageTypeDecline:
			pool.handleDecline(macAddress.String())
			return
		case dhcpv4.MessageTypeRelease:
			pool.handleRelease(macAddress.String())
			return
		default:
			log.Printf("[DHCP Server] 무시하는 메시지 타입: %s", msgType)
			return
		}

		clientHostname := m.HostName()
		ip := pool.handleClientRequest(macAddress.String(), clientHostname, cfg)
		if ip == nil {
			return
		}

		_, ipNet, err := net.ParseCIDR(cfg.Network.Lan.Subnet)
		if err != nil {
			log.Printf("CIDR 파싱 실패: %v", err)
			return
		}

		gatewayIP := net.ParseIP(cfg.Dhcp.Server.Gateway)
		serverIP := net.ParseIP(cfg.Dhcp.Server.Gateway) // 서버 IP = 게이트웨이

		reply, err := dhcpv4.NewReplyFromRequest(
			m,
			dhcpv4.WithYourIP(ip),
			dhcpv4.WithServerIP(serverIP),
			dhcpv4.WithOption(dhcpv4.OptServerIdentifier(serverIP)),
			dhcpv4.WithLeaseTime(cfg.Dhcp.Server.LeaseTime),
			dhcpv4.WithNetmask(ipNet.Mask),
			dhcpv4.WithMessageType(replyType),
			dhcpv4.WithDNS(net.ParseIP(cfg.Dhcp.Server.Dns)),
			dhcpv4.WithRouter(gatewayIP),
		)
		if err != nil {
			log.Printf("reply 생성 실패: %v", err)
			return
		}
		// 브로드캐스트로 응답 전송 (유니캐스트 시 커널 ARP 테이블 오염 방지)
		broadcastAddr := &net.UDPAddr{IP: net.IPv4bcast, Port: 68}
		_, err = conn.WriteTo(reply.ToBytes(), broadcastAddr)
		if err != nil {
			log.Printf("전송 실패: %v", err)
			return
		}
		log.Printf("[DHCP Server] 응답 전송 완료: MAC=%s → IP=%s (타입: %s)", macAddress, ip, reply.MessageType())
	}

	var serverError error
	server, serverError := server4.NewServer(iface, nil, handler)
	defer func(server *server4.Server) {
		err := server.Close()
		if err != nil {
			serverError = err
		}
	}(server)
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve()
	}()

	select {
	case <-ctx.Done():
		return serverError
	case err := <-errCh:
		return err
	}
}
