package main

import (
	"fmt"
	"golang.org/x/net/context"
	"home-router/dhcp"
	"home-router/internal/config"
	"home-router/internal/iface"
	"home-router/nat"
	"home-router/network"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// 1. config 읽기
	cfg := config.GetConfig()
	// 2. LAN IP 설정
	lanIface, err := iface.FindInterfaceByMac(cfg.Network.Lan.MacAddress)
	if err != nil {
		log.Fatalf("LAN 인터페이스를 찾을 수 없습니다: %v", err)
	}

	err = network.SetIP(cfg.Network.Lan.MacAddress, cfg.Network.Lan.Subnet)
	if err != nil {
		log.Fatalf("LAN IP 설정에 실패했습니다: %v", err)
	}

	pool := dhcp.NewPool(
		net.ParseIP(cfg.Dhcp.Server.RangeStart),
		net.ParseIP(cfg.Dhcp.Server.RangeEnd),
	)
	wanIface, err := iface.FindInterfaceByMac(cfg.Network.Wan.MacAddress)

	if err != nil {
		log.Fatalf("WAN 인터페이스를 찾을 수 없습니다: %v", err)
	}

	// 3. NAT 활성화
	err = nat.Enable(wanIface.Attrs().Name)
	if err != nil {
		log.Fatalf("NAT 활성화에 실패했습니다: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
	}()

	// 4. DHCP 서버 시작
	go func() {
		err = dhcp.RunServer(ctx, lanIface.Attrs().Name, pool, cfg)
		if err != nil {
			log.Fatalf("DHCP 서버 시작에 실패했습니다: %v", err)
		}
	}()
	// 5. DHCP 클라이언트 시작 (WAN IP 받기)
	client, err := dhcp.RunClient(wanIface.Attrs().Name, ctx)
	if err != nil {
		log.Fatalf("DHCP 클라이언트 시작에 실패했습니다: %v", err)
	}

	for lease := range client {
		prefixLength, _ := lease.ACK.SubnetMask().Size()
		assignedIP := lease.ACK.YourIPAddr

		cidr := fmt.Sprintf("%s/%d", assignedIP, prefixLength)
		err := network.SetIP(cfg.Network.Wan.MacAddress, cidr)

		if err != nil {
			log.Fatalf("새로운 IP WAN 인터페이스 업데이트에 실패했습니다: %v", err)
		}

		err = nat.Disable(wanIface.Attrs().Name)
		if err != nil {
			log.Fatalf("NAT 갱신에 실패했습니다. %w", err)
		}

		err = nat.Enable(wanIface.Attrs().Name)
		if err != nil {
			log.Fatalf("NAT 갱신에 실패했습니다. %w", err)
		}
	}
}
