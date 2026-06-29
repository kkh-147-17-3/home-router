package network

import (
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
	"home-router/internal/iface"
	"net"
)

func SetIP(mac string, cidr string) error {
	link, err := iface.FindInterfaceByMac(mac)
	if err != nil {
		return err
	}

	ip, err := netlink.ParseAddr(cidr)
	if err != nil {
		return err
	}

	// 현재 시스템의 다른 인터페이스에 같은 IP가 남아있으면 제거 (lo 포함)
	allLinks, _ := netlink.LinkList()
	for _, l := range allLinks {
		if l.Attrs().Index == link.Attrs().Index {
			continue
		}
		addrs, _ := netlink.AddrList(l, netlink.FAMILY_V4)
		for _, a := range addrs {
			if a.IPNet.String() == ip.IPNet.String() {
				netlink.AddrDel(l, &a)
			}
		}
	}

	err = netlink.AddrAdd(link, ip)
	if err != nil {
		if err == unix.EEXIST {
			netlink.AddrReplace(link, ip)
		} else {
			return err
		}
	}

	return netlink.LinkSetUp(link)
}

// LinkUp brings the given interface up (e.g. so a DHCP client can send DISCOVER).
func LinkUp(link netlink.Link) error {
	return netlink.LinkSetUp(link)
}

func SetDefaultRoute(mac string, gateway net.IP) error {
	link, err := iface.FindInterfaceByMac(mac)
	if err != nil {
		return err
	}

	// WAN default route 설정 (이미 있으면 교체)
	err = netlink.RouteReplace(&netlink.Route{
		LinkIndex: link.Attrs().Index,
		Gw:        gateway,
	})
	if err != nil {
		return err
	}

	// 다른 인터페이스의 default route 제거 (wlp4s0 등 경쟁 방지)
	routes, _ := netlink.RouteList(nil, netlink.FAMILY_V4)
	for _, r := range routes {
		if r.Dst == nil && r.LinkIndex != link.Attrs().Index {
			netlink.RouteDel(&r)
		}
	}

	return nil
}
