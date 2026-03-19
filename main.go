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
	log.Println("[1/5] 설정 파일 읽는 중...")
	cfg := config.GetConfig()
	log.Println("[1/5] 설정 파일 로드 완료")

	// 2. LAN IP 설정
	log.Println("[2/5] LAN 인터페이스 설정 중...")
	lanIface, err := iface.FindInterfaceByMac(cfg.Network.Lan.MacAddress)
	if err != nil {
		log.Fatalf("LAN 인터페이스를 찾을 수 없습니다: %v", err)
	}

	err = network.SetIP(cfg.Network.Lan.MacAddress, cfg.Network.Lan.Subnet)
	if err != nil {
		log.Fatalf("LAN IP 설정에 실패했습니다: %v", err)
	}
	log.Printf("[2/5] LAN 설정 완료 (인터페이스: %s, 서브넷: %s)", lanIface.Attrs().Name, cfg.Network.Lan.Subnet)

	pool := dhcp.NewPool(
		net.ParseIP(cfg.Dhcp.Server.RangeStart),
		net.ParseIP(cfg.Dhcp.Server.RangeEnd),
	)

	log.Println("[3/5] WAN 인터페이스 찾는 중...")
	wanIface, err := iface.FindInterfaceByMac(cfg.Network.Wan.MacAddress)
	if err != nil {
		log.Fatalf("WAN 인터페이스를 찾을 수 없습니다: %v", err)
	}

	// 3. NAT 활성화
	log.Printf("[3/5] NAT 활성화 중 (인터페이스: %s)...", wanIface.Attrs().Name)
	err = nat.Enable(wanIface.Attrs().Name, lanIface.Attrs().Name)
	if err != nil {
		log.Fatalf("NAT 활성화에 실패했습니다: %v", err)
	}
	log.Println("[3/5] NAT 활성화 완료")

	// 포트포워딩 설정 (WAN IP 미확보 상태 — 외부 트래픽만)
	for _, pf := range cfg.PortForwarding {
		err = nat.AddPortForward(wanIface.Attrs().Name, "", nat.PortForward{
			Name:         pf.Name,
			Protocol:     pf.Protocol,
			ExternalPort: pf.ExternalPort,
			InternalIP:   pf.InternalIP,
			InternalPort: pf.InternalPort,
		})
		if err != nil {
			log.Printf("포트포워딩 설정 실패: %v", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
	}()

	// 4. DHCP 서버 시작 (인터페이스 복구 자동 재시작)
	log.Printf("[4/5] DHCP 서버 시작 중 (인터페이스: %s, 범위: %s ~ %s)...", lanIface.Attrs().Name, cfg.Dhcp.Server.RangeStart, cfg.Dhcp.Server.RangeEnd)
	go func() {
		lanName := lanIface.Attrs().Name
		for {
			err := dhcp.RunServer(ctx, lanName, pool, cfg)
			if ctx.Err() != nil {
				return
			}
			log.Printf("[DHCP Server] 종료: %v, 인터페이스 복구 대기 중...", err)

			newLan, waitErr := iface.WaitForInterface(cfg.Network.Lan.MacAddress, ctx)
			if waitErr != nil {
				return
			}
			lanName = newLan.Attrs().Name

			if setErr := network.SetIP(cfg.Network.Lan.MacAddress, cfg.Network.Lan.Subnet); setErr != nil {
				log.Printf("[LAN] IP 재설정 실패: %v", setErr)
			}
			log.Printf("[DHCP Server] 인터페이스 복구됨 (%s), 재시작", lanName)
		}
	}()
	log.Println("[4/5] DHCP 서버 시작 완료")

	// 5. DHCP 클라이언트 시작 (WAN IP 받기)
	log.Printf("[5/5] DHCP 클라이언트 시작 중 (인터페이스: %s)...", wanIface.Attrs().Name)
	client, err := dhcp.RunClient(wanIface.Attrs().Name, ctx)
	if err != nil {
		log.Fatalf("DHCP 클라이언트 시작에 실패했습니다: %v", err)
	}
	log.Println("[5/5] DHCP 클라이언트 시작 완료, WAN IP 대기 중...")

	var currentWanIP string
	for lease := range client {
		prefixLength, _ := lease.ACK.SubnetMask().Size()
		assignedIP := lease.ACK.YourIPAddr

		cidr := fmt.Sprintf("%s/%d", assignedIP, prefixLength)

		if cidr == currentWanIP {
			log.Printf("WAN IP 갱신 완료 (변경 없음: %s)", cidr)
			continue
		}
		log.Printf("WAN IP 변경: %s → %s", currentWanIP, cidr)

		err := network.SetIP(cfg.Network.Wan.MacAddress, cidr)
		if err != nil {
			log.Fatalf("새로운 IP WAN 인터페이스 업데이트에 실패했습니다: %v", err)
		}
		log.Printf("WAN 인터페이스 IP 설정 완료: %s", cidr)

		// 기본 라우트 설정
		routers := lease.ACK.Router()
		if len(routers) > 0 {
			err = network.SetDefaultRoute(cfg.Network.Wan.MacAddress, routers[0])
			if err != nil {
				log.Printf("기본 라우트 설정 실패: %v", err)
			} else {
				log.Printf("기본 라우트 설정 완료: via %s", routers[0])
			}
		}

		err = nat.Disable(wanIface.Attrs().Name, lanIface.Attrs().Name)
		if err != nil {
			log.Fatalf("NAT 갱신에 실패했습니다. %w", err)
		}

		err = nat.Enable(wanIface.Attrs().Name, lanIface.Attrs().Name)
		if err != nil {
			log.Fatalf("NAT 갱신에 실패했습니다. %w", err)
		}
		log.Printf("NAT 갱신 완료 (인터페이스: %s)", wanIface.Attrs().Name)

		for _, pf := range cfg.PortForwarding {
			err = nat.AddPortForward(wanIface.Attrs().Name, assignedIP.String(), nat.PortForward{
				Name:         pf.Name,
				Protocol:     pf.Protocol,
				ExternalPort: pf.ExternalPort,
				InternalIP:   pf.InternalIP,
				InternalPort: pf.InternalPort,
			})
			if err != nil {
				log.Printf("포트포워딩 갱신 실패: %v", err)
			}
		}
		currentWanIP = cidr
	}
}
