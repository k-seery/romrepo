package network

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"sort"
	"strings"
	"sync"
	"time"
)

// Device represents a discovered host on the local network.
type Device struct {
	IP       netip.Addr
	Hostname string
	SSHOpen  bool
}

// LocalSubnet returns the /24 prefix of the first non-loopback private IPv4
// address found on the system's network interfaces.
func LocalSubnet() (netip.Prefix, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("listing interfaces: %w", err)
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ip := ipNet.IP.To4()
			if ip == nil {
				continue
			}

			if isPrivate(ip) {
				addr, ok := netip.AddrFromSlice(ip)
				if !ok {
					continue
				}
				return netip.PrefixFrom(addr, 24).Masked(), nil
			}
		}
	}

	return netip.Prefix{}, fmt.Errorf("no private IPv4 interface found")
}

func isPrivate(ip net.IP) bool {
	return ip.IsPrivate()
}

// ScanSubnet probes every address in the given prefix on TCP port 22.
// Hosts that accept or refuse the connection are returned; timeouts are
// excluded. A 50-goroutine semaphore limits concurrency.
func ScanSubnet(ctx context.Context, subnet netip.Prefix) ([]Device, error) {
	var (
		mu      sync.Mutex
		devices []Device
		wg      sync.WaitGroup
		sem     = make(chan struct{}, 50)
	)

	addr := subnet.Addr()
	for {
		if !subnet.Contains(addr) {
			break
		}

		ip := addr
		addr = addr.Next()

		wg.Add(1)
		sem <- struct{}{}

		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			if ctx.Err() != nil {
				return
			}

			target := net.JoinHostPort(ip.String(), "22")
			conn, err := net.DialTimeout("tcp", target, 500*time.Millisecond)

			if conn != nil {
				conn.Close()
				dev := Device{IP: ip, SSHOpen: true}
				dev.Hostname = lookupHost(ip)
				mu.Lock()
				devices = append(devices, dev)
				mu.Unlock()
				return
			}

			if err != nil && isConnectionRefused(err) {
				dev := Device{IP: ip, SSHOpen: false}
				dev.Hostname = lookupHost(ip)
				mu.Lock()
				devices = append(devices, dev)
				mu.Unlock()
			}
			// timeout → no host, skip
		}()
	}

	wg.Wait()

	sort.Slice(devices, func(i, j int) bool {
		if devices[i].SSHOpen != devices[j].SSHOpen {
			return devices[i].SSHOpen
		}
		return devices[i].IP.Less(devices[j].IP)
	})

	return devices, nil
}

func lookupHost(ip netip.Addr) string {
	names, err := net.LookupAddr(ip.String())
	if err != nil || len(names) == 0 {
		return ""
	}
	// Remove trailing dot from FQDN.
	h := names[0]
	if len(h) > 0 && h[len(h)-1] == '.' {
		h = h[:len(h)-1]
	}
	return h
}

func isConnectionRefused(err error) bool {
	if opErr, ok := err.(*net.OpError); ok {
		return opErr.Err != nil && strings.Contains(opErr.Err.Error(), "connection refused")
	}
	return strings.Contains(err.Error(), "connection refused")
}
