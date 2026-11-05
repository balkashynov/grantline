package compare

import (
	"fmt"
	"math/rand"
	"net/netip"
	"testing"
)

func member(a netip.Addr, ps []string) bool {
	for _, s := range ps {
		p, e := netip.ParsePrefix(s)
		if e != nil {
			if a.String() == s {
				return true
			}
		} else if p.Contains(a) {
			return true
		}
	}
	return false
}
func TestNetworkDifferenceAgainstMembershipOracle(t *testing.T) {
	rng := rand.New(rand.NewSource(701))
	for trial := 0; trial < 3000; trial++ {
		lists := [2][]string{}
		for j := range lists {
			for k := 0; k < 1+rng.Intn(12); k++ {
				bits := 24 + rng.Intn(9)
				lists[j] = append(lists[j], fmt.Sprintf("192.0.2.%d/%d", rng.Intn(256), bits))
			}
		}
		a, _ := networks(lists[0], true)
		b, _ := networks(lists[1], true)
		delta := describe(difference(a, b))
		out := []string{}
		for _, d := range delta {
			out = append(out, d.CIDR)
			if !member(netip.MustParseAddr(d.Witness), lists[0]) || member(netip.MustParseAddr(d.Witness), lists[1]) {
				t.Fatal("invalid witness")
			}
		}
		for n := 0; n < 256; n++ {
			ip := netip.MustParseAddr(fmt.Sprintf("192.0.2.%d", n))
			want := member(ip, lists[0]) && !member(ip, lists[1])
			if member(ip, out) != want {
				t.Fatalf("trial %d address %s: incorrect difference", trial, ip)
			}
		}
	}
}
func TestNetworkBoundaries(t *testing.T) {
	cases := []struct {
		a, b []string
		want []string
	}{
		{[]string{"192.0.2.0/24"}, []string{"192.0.2.0/25", "192.0.2.128/25"}, []string{}},
		{[]string{"192.0.2.0/24"}, []string{"192.0.2.0/25", "192.0.2.128/26", "192.0.2.192/27", "192.0.2.224/28", "192.0.2.240/29", "192.0.2.248/30", "192.0.2.252/31", "192.0.2.254/32"}, []string{"192.0.2.255/32"}},
		{[]string{"::/0"}, []string{"::/1", "8000::/1"}, []string{}},
		{[]string{"ffff:ffff:ffff:ffff:ffff:ffff:ffff:fffe/127"}, []string{"ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff/128"}, []string{"ffff:ffff:ffff:ffff:ffff:ffff:ffff:fffe/128"}},
		{[]string{"0.0.0.0/0", "::/0"}, []string{"0.0.0.0/1", "128.0.0.0/1"}, []string{"::/0"}},
	}
	for _, c := range cases {
		a, e := networks(c.a, true)
		if e != nil {
			t.Fatal(e)
		}
		b, _ := networks(c.b, true)
		got := describe(difference(a, b))
		if len(got) != len(c.want) {
			t.Fatalf("got %v want %v", got, c.want)
		}
		for i := range got {
			if got[i].CIDR != c.want[i] {
				t.Fatal(got, c.want)
			}
		}
	}
}
func TestEmptyPolicy(t *testing.T) {
	closed, _ := networks(nil, true)
	open, _ := networks(nil, false)
	if len(closed) != 0 || len(open) != 2 {
		t.Fatal("incorrect empty semantics")
	}
	d := describe(open)
	if d[0].Addresses != "4294967296" || d[1].Addresses != "340282366920938463463374607431768211456" {
		t.Fatal(d)
	}
}
func TestRejectMappedIPv6(t *testing.T) {
	if _, e := networks([]string{"::ffff:192.0.2.1/128"}, true); e == nil {
		t.Fatal("expected unsupported mapped IPv6")
	}
}
