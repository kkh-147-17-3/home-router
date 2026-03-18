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
		log.Fatalln("LAN 인터페이스를 찾을 수 없습니다.")
	}

	err = network.SetIP(cfg.Network.Lan.MacAddress, cfg.Network.Lan.Subnet)
	if err != nil {
		log.Fatalln("LAN IP 설정에 실패했습니다.")
	}

	pool := dhcp.NewPool(
		net.ParseIP(cfg.Dhcp.Server.RangeStart),
		net.ParseIP(cfg.Dhcp.Server.RangeEnd),
	)
	wanIface, err := iface.FindInterfaceByMac(cfg.Network.Wan.MacAddress)

	if err != nil {
		log.Fatalln("WAN 인터페이스를 찾을 수 없습니다.")
	}

	// 3. NAT 활성화
	err = nat.Enable(wanIface.Attrs().Name)
	if err != nil {
		log.Fatalln("NAT 활성화에 실패했습니다.")
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
			log.Fatalln("DHCP 서버 시작에 실패했습니다.")
		}
	}()
	// 5. DHCP 클라이언트 시작 (WAN IP 받기)
	client, err := dhcp.RunClient(wanIface.Attrs().Name, ctx)
	if err != nil {
		log.Fatalln("DHCP 클라이언트 시작에 실패했습니다.")
	}

	for lease := range client {
		prefixLength, _ := lease.ACK.SubnetMask().Size()
		assignedIP := lease.ACK.YourIPAddr

		cidr := fmt.Sprintf("%s/%d", assignedIP, prefixLength)
		err := network.SetIP(cfg.Network.Wan.MacAddress, cidr)

		if err != nil {
			log.Fatalln("새로운 IP WAN 인터페이스 업데이트에 실패했습니다.")
		}

		err = nat.Disable(wanIface.Attrs().Name)
		if err != nil {
			log.Fatalln("NAT 갱신에 실패했습니다. %w", err)
		}

		err = nat.Enable(wanIface.Attrs().Name)
		if err != nil {
			log.Fatalln("NAT 갱신에 실패했습니다. %w", err)
		}
	}
}
