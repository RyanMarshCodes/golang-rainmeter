package sysinfo

import (
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/net"
)

var (
	netMu       sync.Mutex
	netLastSent uint64
	netLastRecv uint64
	netLastAt   time.Time
	netHavePrev bool
)

func isLoopbackNIC(name string) bool {
	n := strings.ToLower(name)
	return n == "lo" || n == "lo0" ||
		strings.Contains(n, "loopback") ||
		strings.HasPrefix(n, "isatap") ||
		strings.HasPrefix(n, "teredo")
}

// NetworkRates returns outbound (up) and inbound (down) bytes/sec across non-loopback NICs.
func NetworkRates() (upBps, downBps float64, ok bool) {
	counters, err := net.IOCounters(true)
	if err != nil || len(counters) == 0 {
		return 0, 0, false
	}
	var sent, recv uint64
	for _, c := range counters {
		if isLoopbackNIC(c.Name) {
			continue
		}
		sent += c.BytesSent
		recv += c.BytesRecv
	}

	now := time.Now()
	netMu.Lock()
	defer netMu.Unlock()
	if !netHavePrev || now.Before(netLastAt) {
		netLastSent, netLastRecv = sent, recv
		netLastAt = now
		netHavePrev = true
		return 0, 0, true // first sample primes the counter
	}
	dt := now.Sub(netLastAt).Seconds()
	if dt < 0.05 {
		return 0, 0, true
	}
	var up, down float64
	if sent >= netLastSent {
		up = float64(sent-netLastSent) / dt
	}
	if recv >= netLastRecv {
		down = float64(recv-netLastRecv) / dt
	}
	netLastSent, netLastRecv = sent, recv
	netLastAt = now
	return up, down, true
}
