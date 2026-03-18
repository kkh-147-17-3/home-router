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
		// 1. 클라이언트 MAC 주소 가져오기
		macAddress := m.ClientHWAddr
		ip := pool.handleClientRequest(macAddress.String(), cfg)
		if ip == nil {
			return
		}
		// 2. handleClientRequest로 IP 할당

		_, ipNet, err := net.ParseCIDR(cfg.Network.Lan.Subnet)
		if err != nil {
			log.Printf("CIDR 파싱 실패: %v", err)
			return
		}

		reply, err := dhcpv4.NewReplyFromRequest(
			m,
			dhcpv4.WithYourIP(ip),
			dhcpv4.WithGatewayIP(net.ParseIP(cfg.Dhcp.Server.Gateway)),
			dhcpv4.WithLeaseTime(cfg.Dhcp.Server.LeaseTime),
			dhcpv4.WithNetmask(ipNet.Mask),
			dhcpv4.WithMessageType(dhcpv4.MessageTypeOffer),
			dhcpv4.WithDNS(net.ParseIP(cfg.Dhcp.Server.Dns)),
		)
		if err != nil {
			log.Printf("reply 생성 실패: %v", err)
			return
		}
		// 3. 응답 전송
		_, err = conn.WriteTo(reply.ToBytes(), peer)
		if err != nil {
			log.Printf("전송 실패: %v", err)
			return
		}
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
