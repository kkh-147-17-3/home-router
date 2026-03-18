package dhcp

import (
	"github.com/insomniacslk/dhcp/dhcpv4/nclient4"
	"golang.org/x/net/context"
	"time"
)

func RunClient(iface string, ctx context.Context) (<-chan *nclient4.Lease, error) {
	ch := make(chan *nclient4.Lease)

	go func() {
		count := 1
		maxCount := 10
		for {
			lease, err := doDHCP(ctx, iface)
			if err != nil {
				if count < maxCount {
					count++

					select {
					case <-time.After(time.Second * 3):
					case <-ctx.Done():
						close(ch)
						return
					}
					continue
				} else {
					close(ch)
					return
				}
			}
			ch <- lease
			renewalTime := lease.ACK.IPAddressLeaseTime(0) / 2
			select {

			case <-time.After(renewalTime):
			case <-ctx.Done():
				close(ch)
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
