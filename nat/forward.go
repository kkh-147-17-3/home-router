package nat

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

const chainName = "HOME-ROUTER"

func Enable(wanIface, lanIface string) error {
	// 1. IP 포워딩 활성화
	err := os.WriteFile("/proc/sys/net/ipv4/ip_forward", []byte("1"), 0644)
	if err != nil {
		return err
	}

	// 2. 커스텀 체인 생성 (Docker FORWARD DROP 정책 우회)
	exec.Command("iptables", "-N", chainName).Run()
	exec.Command("iptables", "-F", chainName).Run()

	// FORWARD 체인 맨 앞에 커스텀 체인으로 점프
	exec.Command("iptables", "-D", "FORWARD", "-j", chainName).Run()
	err = exec.Command("iptables", "-I", "FORWARD", "1", "-j", chainName).Run()
	if err != nil {
		return fmt.Errorf("FORWARD → %s 점프 룰 추가 실패: %w", chainName, err)
	}

	// 3. LAN → WAN 포워딩 허용
	err = exec.Command("iptables", "-A", chainName,
		"-i", lanIface, "-o", wanIface, "-j", "ACCEPT").Run()
	if err != nil {
		return fmt.Errorf("LAN→WAN FORWARD 룰 추가 실패: %w", err)
	}

	// 4. WAN → LAN 응답 트래픽 허용 (RELATED,ESTABLISHED)
	err = exec.Command("iptables", "-A", chainName,
		"-i", wanIface, "-o", lanIface,
		"-m", "state", "--state", "RELATED,ESTABLISHED",
		"-j", "ACCEPT").Run()
	if err != nil {
		return fmt.Errorf("WAN→LAN FORWARD 룰 추가 실패: %w", err)
	}

	// 4-1. LAN → LAN 포워딩 허용 (Hairpin NAT)
	err = exec.Command("iptables", "-A", chainName,
		"-i", lanIface, "-o", lanIface, "-j", "ACCEPT").Run()
	if err != nil {
		return fmt.Errorf("LAN→LAN FORWARD 룰 추가 실패: %w", err)
	}

	// 5. MASQUERADE (NAT)
	exec.Command("iptables", "-t", "nat", "-D", "POSTROUTING",
		"-o", wanIface, "-j", "MASQUERADE").Run()
	err = exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING",
		"-o", wanIface, "-j", "MASQUERADE").Run()
	if err != nil {
		return fmt.Errorf("MASQUERADE 룰 추가 실패: %w", err)
	}

	// 6. INPUT: LAN에서 라우터로의 트래픽 허용
	exec.Command("iptables", "-D", "INPUT", "-i", lanIface, "-j", "ACCEPT").Run()
	err = exec.Command("iptables", "-I", "INPUT", "-i", lanIface, "-j", "ACCEPT").Run()
	if err != nil {
		return fmt.Errorf("INPUT 룰 추가 실패: %w", err)
	}

	// 7. TCP MSS Clamping (MTU 불일치로 인한 패킷 드랍 방지)
	exec.Command("iptables", "-t", "mangle", "-D", "FORWARD",
		"-p", "tcp", "--tcp-flags", "SYN,RST", "SYN",
		"-j", "TCPMSS", "--clamp-mss-to-pmtu").Run()
	err = exec.Command("iptables", "-t", "mangle", "-A", "FORWARD",
		"-p", "tcp", "--tcp-flags", "SYN,RST", "SYN",
		"-j", "TCPMSS", "--clamp-mss-to-pmtu").Run()
	if err != nil {
		return fmt.Errorf("MSS clamping 룰 추가 실패: %w", err)
	}

	return nil
}

type PortForward struct {
	Name         string
	Protocol     string
	ExternalPort int
	InternalIP   string
	InternalPort int
}

func AddPortForward(wanIface string, wanIP string, pf PortForward) error {
	extPort := fmt.Sprintf("%d", pf.ExternalPort)
	intPort := fmt.Sprintf("%d", pf.InternalPort)
	dest := fmt.Sprintf("%s:%d", pf.InternalIP, pf.InternalPort)

	// DNAT: WAN 인터페이스에서 들어오는 외부 트래픽
	exec.Command("iptables", "-t", "nat", "-D", "PREROUTING",
		"-i", wanIface, "-p", pf.Protocol, "--dport", extPort,
		"-j", "DNAT", "--to-destination", dest).Run()
	err := exec.Command("iptables", "-t", "nat", "-I", "PREROUTING",
		"-i", wanIface, "-p", pf.Protocol, "--dport", extPort,
		"-j", "DNAT", "--to-destination", dest).Run()
	if err != nil {
		return fmt.Errorf("DNAT 룰 추가 실패 (%s): %w", pf.Name, err)
	}

	// Hairpin DNAT: LAN에서 WAN IP로 접근하는 트래픽
	if wanIP != "" {
		exec.Command("iptables", "-t", "nat", "-D", "PREROUTING",
			"-d", wanIP, "-p", pf.Protocol, "--dport", extPort,
			"-j", "DNAT", "--to-destination", dest).Run()
		err = exec.Command("iptables", "-t", "nat", "-I", "PREROUTING",
			"-d", wanIP, "-p", pf.Protocol, "--dport", extPort,
			"-j", "DNAT", "--to-destination", dest).Run()
		if err != nil {
			return fmt.Errorf("Hairpin DNAT 룰 추가 실패 (%s): %w", pf.Name, err)
		}

		// Hairpin MASQUERADE: 응답이 라우터를 거치도록
		exec.Command("iptables", "-t", "nat", "-D", "POSTROUTING",
			"-s", "192.168.1.0/24", "-d", pf.InternalIP,
			"-p", pf.Protocol, "--dport", intPort,
			"-j", "MASQUERADE").Run()
		err = exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING",
			"-s", "192.168.1.0/24", "-d", pf.InternalIP,
			"-p", pf.Protocol, "--dport", intPort,
			"-j", "MASQUERADE").Run()
		if err != nil {
			return fmt.Errorf("Hairpin NAT 룰 추가 실패 (%s): %w", pf.Name, err)
		}
	}

	// FORWARD: 포워딩된 트래픽 허용 (커스텀 체인)
	exec.Command("iptables", "-D", chainName,
		"-p", pf.Protocol, "-d", pf.InternalIP, "--dport", intPort,
		"-j", "ACCEPT").Run()
	err = exec.Command("iptables", "-I", chainName,
		"-p", pf.Protocol, "-d", pf.InternalIP, "--dport", intPort,
		"-j", "ACCEPT").Run()
	if err != nil {
		return fmt.Errorf("FORWARD 룰 추가 실패 (%s): %w", pf.Name, err)
	}

	log.Printf("포트포워딩 설정 완료: %s (%s:%s → %s)", pf.Name, pf.Protocol, extPort, dest)
	return nil
}

func RemovePortForward(wanIface string, wanIP string, pf PortForward) {
	extPort := fmt.Sprintf("%d", pf.ExternalPort)
	intPort := fmt.Sprintf("%d", pf.InternalPort)
	dest := fmt.Sprintf("%s:%d", pf.InternalIP, pf.InternalPort)

	exec.Command("iptables", "-t", "nat", "-D", "PREROUTING",
		"-i", wanIface, "-p", pf.Protocol, "--dport", extPort,
		"-j", "DNAT", "--to-destination", dest).Run()
	if wanIP != "" {
		exec.Command("iptables", "-t", "nat", "-D", "PREROUTING",
			"-d", wanIP, "-p", pf.Protocol, "--dport", extPort,
			"-j", "DNAT", "--to-destination", dest).Run()
		exec.Command("iptables", "-t", "nat", "-D", "POSTROUTING",
			"-s", "192.168.1.0/24", "-d", pf.InternalIP,
			"-p", pf.Protocol, "--dport", intPort,
			"-j", "MASQUERADE").Run()
	}
	exec.Command("iptables", "-D", chainName,
		"-p", pf.Protocol, "-d", pf.InternalIP, "--dport", intPort,
		"-j", "ACCEPT").Run()
}

func Disable(wanIface, lanIface string) error {
	// 커스텀 체인 정리
	exec.Command("iptables", "-D", "FORWARD", "-j", chainName).Run()
	exec.Command("iptables", "-F", chainName).Run()
	exec.Command("iptables", "-X", chainName).Run()

	// MASQUERADE 제거
	exec.Command("iptables", "-t", "nat", "-D", "POSTROUTING",
		"-o", wanIface, "-j", "MASQUERADE").Run()

	// INPUT 제거
	exec.Command("iptables", "-D", "INPUT", "-i", lanIface, "-j", "ACCEPT").Run()

	// MSS Clamping 제거
	exec.Command("iptables", "-t", "mangle", "-D", "FORWARD",
		"-p", "tcp", "--tcp-flags", "SYN,RST", "SYN",
		"-j", "TCPMSS", "--clamp-mss-to-pmtu").Run()

	return nil
}
