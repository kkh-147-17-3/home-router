package iface

import (
	"fmt"
	"github.com/vishvananda/netlink"
	"golang.org/x/net/context"
	"log"
	"time"
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

func WaitForInterface(mac string, ctx context.Context) (netlink.Link, error) {
	for {
		link, err := FindInterfaceByMac(mac)
		if err == nil {
			return link, nil
		}
		log.Printf("[Interface] MAC %s 인터페이스 대기 중...", mac)
		select {
		case <-time.After(3 * time.Second):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}
