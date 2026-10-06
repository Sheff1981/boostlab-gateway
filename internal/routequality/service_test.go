package routequality

import (
	"net"
	"testing"
	"time"
)

func TestParseTargets(t *testing.T) {
	targets, err := ParseTargets(`[
		{"id":"pubg-eu","host":"example.com","tcp_port":443},
		{"id":"cod-eu","host":"1.1.1.1","tcp_port":443}
	]`)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 2 || targets[0].ID != "pubg-eu" {
		t.Fatalf("unexpected targets: %#v", targets)
	}
}

func TestParseTargetsRejectsDuplicate(t *testing.T) {
	_, err := ParseTargets(`[
		{"id":"same","host":"example.com","tcp_port":443},
		{"id":"SAME","host":"example.org","tcp_port":443}
	]`)
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestCalculateMedianP95JitterLoss(t *testing.T) {
	a, b, c, d := 20, 40, 30, 50
	metrics := calculate(
		"target",
		[]*int{&a, &b, nil, &c, &d},
		time.Unix(1_700_000_000, 0),
	)
	if metrics.MedianRTTMs == nil || *metrics.MedianRTTMs != 35 {
		t.Fatalf("unexpected median: %#v", metrics.MedianRTTMs)
	}
	if metrics.P95RTTMs == nil || *metrics.P95RTTMs != 50 {
		t.Fatalf("unexpected p95: %#v", metrics.P95RTTMs)
	}
	if metrics.JitterMs == nil || *metrics.JitterMs != 17 {
		t.Fatalf("unexpected jitter: %#v", metrics.JitterMs)
	}
	if metrics.PacketLossPct != 20 {
		t.Fatalf("unexpected loss: %v", metrics.PacketLossPct)
	}
}

func TestFirstPublicIPRejectsPrivate(t *testing.T) {
	ip, ok := firstPublicIP([]net.IP{
		net.ParseIP("127.0.0.1"),
		net.ParseIP("10.0.0.1"),
		net.ParseIP("1.1.1.1"),
	})
	if !ok || !ip.Equal(net.ParseIP("1.1.1.1")) {
		t.Fatalf("unexpected public IP: %v ok=%v", ip, ok)
	}
}
