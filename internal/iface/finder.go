package iface

import (
	"fmt"
	"github.com/vishvananda/netlink"
)

func FindInterfaceByMac(mac string) (netlink.Link, error) {
	links, err := netlink.LinkList() // 모든 인터페이스 가져오기
	if err != nil {
		return nil, err
	}

	for _, link := range links {
		if link.Attrs().HardwareAddr.String() == mac {
			return link, nil
		}
	}

	return nil, fmt.Errorf("인터페이스를 찾을 수 없음: %s", mac)
}
