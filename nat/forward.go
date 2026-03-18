package nat

import (
	"os"
	"os/exec"
)

func Enable(wanIface string) error {
	// 1. IP 포워딩 활성화
	err := os.WriteFile("/proc/sys/net/ipv4/ip_forward", []byte("1"), 0644)
	if err != nil {
		return err
	}

	err = exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING", "-o", wanIface, "-j", "MASQUERADE").Run()
	if err != nil {
		return err
	}

	return nil
}

func Disable(wanIface string) error {
	err := exec.Command("iptables", "-t", "nat", "-D", "POSTROUTING", "-o", wanIface, "-j", "MASQUERADE").Run()
	if err != nil {
		return err
	}

	return nil
}
