package hostip

import (
	"net"
	"strings"
)

func Resolve(configured string) string {
	trimmed := strings.TrimSpace(configured)
	if !isLoopbackHost(trimmed) {
		return trimmed
	}
	if ip := firstPrivateIPv4(); ip != "" {
		return ip
	}
	if trimmed != "" {
		return trimmed
	}
	return "127.0.0.1"
}

func isLoopbackHost(host string) bool {
	switch strings.ToLower(strings.TrimSpace(host)) {
	case "", "127.0.0.1", "localhost", "::1", "0.0.0.0":
		return true
	default:
		return false
	}
}

func firstPrivateIPv4() string {
	if ip := outboundIPv4(); ip != "" {
		return ip
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	var classA, classC, classB []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if shouldSkipInterface(iface.Name) {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip := ipFromAddr(addr)
			if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}
			v4 := ip.To4()
			if v4 == nil || !v4.IsPrivate() {
				continue
			}
			s := v4.String()
			switch {
			case v4[0] == 10:
				classA = append(classA, s)
			case v4[0] == 192 && v4[1] == 168:
				if v4[2] == 56 || v4[2] == 137 || v4[2] == 192 {
					continue
				}
				classC = append(classC, s)
			case v4[0] == 172 && v4[1] >= 16 && v4[1] <= 31:
				if v4[1] == 17 || v4[1] == 18 || v4[1] == 19 || v4[1] == 26 {
					continue
				}
				classB = append(classB, s)
			}
		}
	}
	if len(classA) > 0 {
		return classA[0]
	}
	if len(classC) > 0 {
		return classC[0]
	}
	if len(classB) > 0 {
		return classB[0]
	}
	return ""
}

func outboundIPv4() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()
	udp, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok || udp.IP == nil {
		return ""
	}
	v4 := udp.IP.To4()
	if v4 == nil || v4.IsLoopback() || !v4.IsPrivate() {
		return ""
	}
	return v4.String()
}

func shouldSkipInterface(name string) bool {
	n := strings.ToLower(name)
	skip := []string{
		"loopback", "wsl", "docker", "hyper-v", "default switch",
		"virtualbox", "vmware", "bluetooth", "isatap", "teredo",
	}
	for _, s := range skip {
		if strings.Contains(n, s) {
			return true
		}
	}
	return false
}

func ipFromAddr(addr net.Addr) net.IP {
	switch v := addr.(type) {
	case *net.IPNet:
		return v.IP
	case *net.IPAddr:
		return v.IP
	default:
		return nil
	}
}
