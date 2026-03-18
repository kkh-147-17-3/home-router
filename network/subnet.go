package network

import (
	"github.com/vishvananda/netlink"
	"home-router/internal/iface"
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

	err = netlink.AddrAdd(link, ip)
	if err != nil {
		return err
	}

	return netlink.LinkSetUp(link)
}
