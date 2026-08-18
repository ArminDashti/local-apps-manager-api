package probe

import (
	"net"
	"strconv"
	"time"
)

func PortListening(host string, port int) bool {
	if port <= 0 {
		return false
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
