package dhcp

import (
	"github.com/insomniacslk/dhcp/dhcpv4/nclient4"
	"golang.org/x/net/context"
	"log"
	"time"
)

func RunClient(iface string, ctx context.Context) (<-chan *nclient4.Lease, error) {
	ch := make(chan *nclient4.Lease)

	go func() {
		defer close(ch)
		retryDelay := 3 * time.Second
		maxDelay := 60 * time.Second

		for {
			lease, err := doDHCP(ctx, iface)

			if err != nil {
				log.Printf("[DHCP Client] 요청 실패: %v, %s 후 재시도", err, retryDelay)
				select {
				case <-time.After(retryDelay):
					retryDelay = retryDelay * 2
					if retryDelay > maxDelay {
						retryDelay = maxDelay
					}
				case <-ctx.Done():
					return
				}
				continue
			}
			retryDelay = 3 * time.Second
			ch <- lease

			renewalTime := lease.ACK.IPAddressLeaseTime(0) / 2
			if renewalTime < 30*time.Second {
				renewalTime = 30 * time.Second
			}
			log.Printf("[DHCP Client] 다음 갱신: %s 후", renewalTime)
			select {
			case <-time.After(renewalTime):
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}

func doDHCP(ctx context.Context, iface string) (*nclient4.Lease, error) {
	client, err := nclient4.New(iface)
	defer func() {
		if client == nil {
			return
		}
		if closeErr := client.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	if err != nil {
		return nil, err
	}

	lease, err := client.Request(ctx)
	if err != nil {
		return nil, err
	}

	return lease, nil
}
