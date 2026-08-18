package hostip

import (
	"net"
	"testing"
)

func TestResolveUsesConfiguredLanIP(t *testing.T) {
	if got := Resolve("10.20.9.59"); got != "10.20.9.59" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveTreatsLoopbackAsNeedDetect(t *testing.T) {
	if isLoopbackHost("127.0.0.1") != true {
		t.Fatal("127.0.0.1 should be loopback")
	}
	if isLoopbackHost("10.20.9.59") {
		t.Fatal("LAN IP should not be loopback")
	}
}

func TestResolveDetectsPrivateIPv4(t *testing.T) {
	got := Resolve("127.0.0.1")
	ip := net.ParseIP(got)
	if ip == nil || ip.To4() == nil {
		t.Fatalf("expected IPv4, got %q", got)
	}
	t.Logf("detected %s", got)
}

func TestIsLoopbackHost(t *testing.T) {
	cases := map[string]bool{
		"":          true,
		"localhost": true,
		"::1":       true,
		"0.0.0.0":   true,
		"10.1.1.1":  false,
	}
	for in, want := range cases {
		if got := isLoopbackHost(in); got != want {
			t.Fatalf("%q: got %v want %v", in, got, want)
		}
	}
}
